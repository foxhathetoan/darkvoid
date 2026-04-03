package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/pkg/logger"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
)

func channelKey(userID uuid.UUID) string {
	return fmt.Sprintf("notif:stream:%s", userID)
}

// Event is a notification event sent to SSE clients.
type Event struct {
	Type string          `json:"type"` // "notification" or "unread_count"
	Data json.RawMessage `json:"data"`
}

// client represents a single SSE connection.
type client struct {
	ch        chan Event
	userID    uuid.UUID
	closeOnce sync.Once
}

func (c *client) close() {
	c.closeOnce.Do(func() { close(c.ch) })
}

// Broker manages SSE client connections and dispatches notification events.
// When Redis is available, it uses Pub/Sub for cross-instance fan-out.
// When Redis is nil, it operates as an in-memory-only broker (single instance).
type Broker struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*client]struct{} // userID → set of clients
	redis   *pkgredis.Client                   // nil = in-memory only
}

// NewBroker creates a new notification broker.
func NewBroker(redis *pkgredis.Client) *Broker {
	return &Broker{
		clients: make(map[uuid.UUID]map[*client]struct{}),
		redis:   redis,
	}
}

// Subscribe registers an SSE client for a user. Returns a channel that receives
// events and a cleanup function that must be called when the connection closes.
func (b *Broker) Subscribe(ctx context.Context, userID uuid.UUID) (<-chan Event, func()) {
	c := &client{
		ch:     make(chan Event, 64),
		userID: userID,
	}

	b.mu.Lock()
	if b.clients[userID] == nil {
		b.clients[userID] = make(map[*client]struct{})
	}
	b.clients[userID][c] = struct{}{}
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		delete(b.clients[userID], c)
		if len(b.clients[userID]) == 0 {
			delete(b.clients, userID)
		}
		b.mu.Unlock()
		c.close()
	}

	return c.ch, cleanup
}

// Shutdown closes all active SSE client channels, causing all Stream handlers
// to return. Must be called before http.Server.Shutdown to avoid the graceful
// shutdown timeout being hit by long-lived SSE connections.
func (b *Broker) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, clients := range b.clients {
		for c := range clients {
			c.close()
		}
	}
}

// Publish sends an event to all SSE clients connected for the given userID.
// If Redis is configured, it publishes to the Pub/Sub channel so other instances
// can forward the event to their local clients.
func (b *Broker) Publish(ctx context.Context, userID uuid.UUID, evt Event) {
	if b.redis != nil {
		raw, err := json.Marshal(evt)
		if err != nil {
			logger.LogError(ctx, err, "failed to marshal SSE event", "user_id", userID)
			return
		}
		if err := b.redis.Publish(ctx, channelKey(userID), raw).Err(); err != nil {
			logger.LogError(ctx, err, "failed to publish SSE event to Redis", "user_id", userID)
		}
		// Redis subscriber (StartRedisSubscriber) will call deliverLocal.
		return
	}

	// No Redis — deliver directly to local clients.
	b.deliverLocal(userID, evt)
}

// deliverLocal fans out an event to all in-memory clients for a user.
func (b *Broker) deliverLocal(userID uuid.UUID, evt Event) {
	b.mu.RLock()
	clients := b.clients[userID]
	b.mu.RUnlock()

	for c := range clients {
		select {
		case c.ch <- evt:
		default:
			// Client buffer full — drop event to avoid blocking
		}
	}
}

// StartRedisSubscriber listens to Redis Pub/Sub and forwards events to local clients.
// It dynamically subscribes/unsubscribes based on connected clients.
// Blocks until ctx is cancelled. Should be run in a goroutine.
func (b *Broker) StartRedisSubscriber(ctx context.Context) {
	if b.redis == nil {
		return
	}

	// Use pattern subscribe: notif:stream:*
	pubsub := b.redis.PSubscribe(ctx, "notif:stream:*")
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Parse channel to extract userID: "notif:stream:<uuid>"
			var userIDStr string
			if len(msg.Channel) > len("notif:stream:") {
				userIDStr = msg.Channel[len("notif:stream:"):]
			}
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				logger.LogError(ctx, err, "invalid user ID in pub/sub channel", "channel", msg.Channel)
				continue
			}

			var evt Event
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				logger.LogError(ctx, err, "failed to unmarshal pub/sub event")
				continue
			}

			b.deliverLocal(userID, evt)
		}
	}
}
