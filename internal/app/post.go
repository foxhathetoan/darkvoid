package app

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/app/middleware"
	postcache "github.com/jarviisha/darkvoid/internal/feature/post/cache"
	postentity "github.com/jarviisha/darkvoid/internal/feature/post/entity"
	"github.com/jarviisha/darkvoid/internal/feature/post/handler"
	"github.com/jarviisha/darkvoid/internal/feature/post/repository"
	"github.com/jarviisha/darkvoid/internal/feature/post/service"
	userrepo "github.com/jarviisha/darkvoid/internal/feature/user/repository"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// postUserReader implements service.userReader using UserRepository.
type postUserReader struct {
	userRepo *userrepo.UserRepository
}

func (r *postUserReader) GetAuthorsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*postentity.Author, error) {
	users, err := r.userRepo.GetUsersByIDsAny(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]*postentity.Author, len(users))
	for _, u := range users {
		result[u.ID] = &postentity.Author{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarKey:   u.AvatarKey,
		}
	}
	return result, nil
}

// PostContext represents the Post bounded context with all its dependencies
type PostContext struct {
	// Repositories
	PostRepo           *repository.PostRepository
	MediaRepo          *repository.MediaRepository
	LikeRepo           *repository.LikeRepository
	CommentRepo        *repository.CommentRepository
	CommentMediaRepo   *repository.CommentMediaRepository
	HashtagRepo        *repository.HashtagRepository
	MentionRepo        *repository.MentionRepository
	CommentMentionRepo *repository.CommentMentionRepository

	// Services
	PostService        *service.PostService
	LikeService        *service.LikeService
	CommentService     *service.CommentService
	CommentLikeService *service.CommentLikeService
	HashtagService     *service.HashtagService

	// Handlers
	PostHandler        *handler.PostHandler
	LikeHandler        *handler.LikeHandler
	CommentHandler     *handler.CommentHandler
	CommentLikeHandler *handler.CommentLikeHandler
	HashtagHandler     *handler.HashtagHandler
}

// SetupPostContext initializes the Post context with all required dependencies.
func SetupPostContext(pool *pgxpool.Pool, store storage.Storage, userRepo *userrepo.UserRepository, redis *pkgredis.Client) *PostContext {
	// Repositories
	postRepo := repository.NewPostRepository(pool)
	mediaRepo := repository.NewMediaRepository(pool)
	likeRepo := repository.NewLikeRepository(pool)
	commentRepo := repository.NewCommentRepository(pool)
	commentMediaRepo := repository.NewCommentMediaRepository(pool)
	commentLikeRepo := repository.NewCommentLikeRepository(pool)
	hashtagRepo := repository.NewHashtagRepository(pool)
	mentionRepo := repository.NewMentionRepository(pool)
	commentMentionRepo := repository.NewCommentMentionRepository(pool)

	ur := &postUserReader{userRepo: userRepo}

	// Hashtag cache — use Redis when available, nop otherwise
	var hCache postcache.HashtagCache
	if redis != nil {
		hCache = postcache.NewRedisHashtagCache(redis)
	} else {
		hCache = postcache.NewNopHashtagCache()
	}

	// Services
	postService := service.NewPostService(pool, postRepo, mediaRepo, ur, hashtagRepo,
		service.WithLikeRepo(likeRepo),
		service.WithMentionRepo(mentionRepo),
	)
	likeService := service.NewLikeService(likeRepo, postRepo)
	commentService := service.NewCommentService(pool, commentRepo, commentMediaRepo, postRepo, ur,
		service.WithCommentLikeRepo(commentLikeRepo),
		service.WithCommentMentionRepo(commentMentionRepo),
	)
	commentLikeService := service.NewCommentLikeService(commentLikeRepo, commentRepo)
	hashtagService := service.NewHashtagService(hashtagRepo, hCache, postRepo, ur)

	// Handlers
	postHandler := handler.NewPostHandler(postService, store)
	likeHandler := handler.NewLikeHandler(likeService)
	commentHandler := handler.NewCommentHandler(commentService, store)
	commentLikeHandler := handler.NewCommentLikeHandler(commentLikeService)
	hashtagHandler := handler.NewHashtagHandler(hashtagService, store)

	return &PostContext{
		PostRepo:           postRepo,
		MediaRepo:          mediaRepo,
		LikeRepo:           likeRepo,
		CommentRepo:        commentRepo,
		CommentMediaRepo:   commentMediaRepo,
		HashtagRepo:        hashtagRepo,
		MentionRepo:        mentionRepo,
		CommentMentionRepo: commentMentionRepo,
		PostService:        postService,
		LikeService:        likeService,
		CommentService:     commentService,
		CommentLikeService: commentLikeService,
		HashtagService:     hashtagService,
		PostHandler:        postHandler,
		LikeHandler:        likeHandler,
		CommentHandler:     commentHandler,
		CommentLikeHandler: commentLikeHandler,
		HashtagHandler:     hashtagHandler,
	}
}

// RegisterRoutes registers all routes for the Post context
func (ctx *PostContext) RegisterRoutes(r chi.Router, auth middleware.AuthMiddleware) {
	// Public routes (no authentication required)
	r.Group(func(r chi.Router) {
		r.Get("/posts/{postID}", ctx.PostHandler.GetPost)
		r.Get("/users/{userID}/posts", ctx.PostHandler.GetUserPosts)
		r.Get("/posts/{postID}/comments", ctx.CommentHandler.GetComments)
		r.Get("/posts/{postID}/comments/{commentID}/replies", ctx.CommentHandler.GetReplies)

		// Hashtag routes — rate limited: 60 req/min per IP
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimitByIP(60, time.Minute))
			r.Get("/hashtags/trending", ctx.HashtagHandler.GetTrendingHashtags)
			r.Get("/hashtags/search", ctx.HashtagHandler.SearchHashtags)
			r.Get("/hashtags/{name}/posts", ctx.HashtagHandler.GetPostsByHashtag)
		})
	})

	// Protected routes (authentication required)
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)

		// Post CRUD
		r.Post("/posts", ctx.PostHandler.CreatePost)
		r.Put("/posts/{postID}", ctx.PostHandler.UpdatePost)
		r.Delete("/posts/{postID}", ctx.PostHandler.DeletePost)

		// Likes
		r.Post("/posts/{postID}/like", ctx.LikeHandler.ToggleLike)

		// Comments
		r.Post("/posts/{postID}/comments", ctx.CommentHandler.CreateComment)
		r.Delete("/posts/{postID}/comments/{commentID}", ctx.CommentHandler.DeleteComment)

		// Comment likes
		r.Post("/posts/{postID}/comments/{commentID}/like", ctx.CommentLikeHandler.ToggleCommentLike)
	})
}
