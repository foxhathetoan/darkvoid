// cmd/seed/main.go — Sandbox data seeder
//
// Usage:
//
//	go run ./cmd/seed [--users=100] [--posts=1000] [--likes-per-post=50] [--reset]
//
// Flags:
//
//	--users           number of users to create (default 100)
//	--posts           number of posts to create (default 1000)
//	--likes-per-post  max likes per post, randomised 0..N (default 50)
//	--reset           delete all seed data before seeding (truncates post+user tables)
//
// The seeder creates realistic data for testing the feed:
//   - Users with unique usernames/emails
//   - Posts distributed across users, with random visibility and age (0–14 days)
//   - Random follows: each user follows ~20% of others
//   - Random likes: each post gets 0..likes-per-post likes from random users
//
// like_count on posts is maintained by the DB trigger from migration 000006.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/pkg/config"
	"github.com/jarviisha/darkvoid/pkg/database"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	users := flag.Int("users", 100, "number of users to create")
	posts := flag.Int("posts", 1000, "number of posts to create")
	likesPerPost := flag.Int("likes-per-post", 50, "max likes per post (randomised 0..N)")
	reset := flag.Bool("reset", false, "truncate seed data before seeding")
	flag.Parse()

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := database.NewPostgresPool(ctx, &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
	})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err = database.HealthCheck(ctx, pool); err != nil {
		log.Fatalf("db health check: %v", err)
	}

	s := &seeder{pool: pool, rng: rand.New(rand.NewSource(time.Now().UnixNano()))} //nolint:gosec // seed data does not need crypto/rand

	if *reset {
		log.Println("resetting seed data...")
		s.reset(ctx)
	}

	log.Printf("seeding %d users, %d posts, up to %d likes/post...", *users, *posts, *likesPerPost)

	userIDs, err := s.seedUsers(ctx, *users)
	if err != nil {
		log.Fatalf("seed users: %v", err)
	}
	log.Printf("  created %d users", len(userIDs))

	if err = s.seedFollows(ctx, userIDs); err != nil {
		log.Fatalf("seed follows: %v", err)
	}
	log.Printf("  created follows (~20%% follow rate)")

	postIDs, err := s.seedPosts(ctx, userIDs, *posts)
	if err != nil {
		log.Fatalf("seed posts: %v", err)
	}
	log.Printf("  created %d posts", len(postIDs))

	if err = s.seedLikes(ctx, userIDs, postIDs, *likesPerPost); err != nil {
		log.Fatalf("seed likes: %v", err)
	}
	log.Printf("  created likes")

	log.Println("done.")
}

type seeder struct {
	pool *pgxpool.Pool
	rng  *rand.Rand
}

// reset truncates post and user tables. Safe to call multiple times.
func (s *seeder) reset(ctx context.Context) {
	// Order matters due to FK constraints.
	stmts := []string{
		`DELETE FROM post.likes`,
		`DELETE FROM post.post_media`,
		`DELETE FROM post.comments`,
		`DELETE FROM post.posts`,
		`DELETE FROM usr.follows`,
		`DELETE FROM usr.users`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			log.Printf("  warn: %s — %v", stmt, err)
		}
	}
}

// seedUsers inserts N users and returns their IDs.
func (s *seeder) seedUsers(ctx context.Context, n int) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, n)

	// Generate once and reuse — bcrypt cost 12 matches the app's bcryptCost.
	// plaintext: "Password123!"
	hash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	seedPasswordHash := string(hash)

	for i := range n {
		id := uuid.New()
		username := fmt.Sprintf("user_%s", id.String()[:8])
		email := fmt.Sprintf("%s@seed.local", username)
		displayName := fmt.Sprintf("Seed User %d", i+1)

		_, err := s.pool.Exec(ctx, `
			INSERT INTO usr.users (id, username, email, password_hash, is_active, display_name)
			VALUES ($1, $2, $3, $4, true, $5)
			ON CONFLICT DO NOTHING`,
			id, username, email, seedPasswordHash, displayName,
		)
		if err != nil {
			return nil, fmt.Errorf("insert user %d: %w", i, err)
		}

		ids = append(ids, id)
	}
	return ids, nil
}

// seedFollows creates follow relationships: each user follows ~20% of other users.
func (s *seeder) seedFollows(ctx context.Context, userIDs []uuid.UUID) error {
	for _, follower := range userIDs {
		for _, followee := range userIDs {
			if follower == followee {
				continue
			}
			if s.rng.Float32() > 0.2 {
				continue
			}
			_, err := s.pool.Exec(ctx, `
				INSERT INTO usr.follows (follower_id, followee_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING`,
				follower, followee,
			)
			if err != nil {
				return fmt.Errorf("insert follow: %w", err)
			}
		}
	}
	return nil
}

// seedPosts creates n posts distributed randomly across users.
// Post ages are randomised 0–14 days to exercise both following and trending windows.
func (s *seeder) seedPosts(ctx context.Context, userIDs []uuid.UUID, n int) ([]uuid.UUID, error) {
	visibilities := []string{"public", "public", "public", "followers"} // 75% public
	ids := make([]uuid.UUID, 0, n)

	for i := range n {
		authorID := userIDs[s.rng.Intn(len(userIDs))]
		visibility := visibilities[s.rng.Intn(len(visibilities))]

		// Age: random between 0 and 14 days
		// About 1/14 of posts (~70) will fall in the 24h trending window
		// Posts within 7 days will be in following window
		hoursAgo := s.rng.Float64() * 14 * 24
		createdAt := time.Now().Add(-time.Duration(hoursAgo * float64(time.Hour)))

		content := fmt.Sprintf("Post #%d — %s", i+1, lorem(s.rng))

		id := uuid.New()
		_, err := s.pool.Exec(ctx, `
			INSERT INTO post.posts (id, author_id, content, visibility, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)`,
			id, authorID, content, visibility, createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert post %d: %w", i, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// seedLikes creates random likes.
// Each post gets between 0 and maxLikes likes from random users.
// The DB trigger updates like_count automatically.
func (s *seeder) seedLikes(ctx context.Context, userIDs, postIDs []uuid.UUID, maxLikes int) error {
	for _, postID := range postIDs {
		n := s.rng.Intn(maxLikes + 1)
		// Pick n distinct users
		perm := s.rng.Perm(len(userIDs))
		if n > len(perm) {
			n = len(perm)
		}
		for _, idx := range perm[:n] {
			_, err := s.pool.Exec(ctx, `
				INSERT INTO post.likes (user_id, post_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING`,
				userIDs[idx], postID,
			)
			if err != nil {
				return fmt.Errorf("insert like: %w", err)
			}
		}
	}
	return nil
}

// lorem returns a short random sentence for post content.
func lorem(rng *rand.Rand) string {
	words := []string{
		"backend", "feed", "ranking", "score", "like", "follow", "trending",
		"algorithm", "decay", "recency", "engagement", "viral", "content",
		"test", "sandbox", "data", "post", "user", "social", "network",
	}
	n := 5 + rng.Intn(10)
	out := make([]string, n)
	for i := range n {
		out[i] = words[rng.Intn(len(words))]
	}
	return fmt.Sprintf("%v", out)
}
