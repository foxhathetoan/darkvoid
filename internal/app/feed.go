package app

import (
	"github.com/go-chi/chi/v5"

	"github.com/jarviisha/darkvoid/internal/app/middleware"
	"github.com/jarviisha/darkvoid/internal/feature/feed"
	feedcache "github.com/jarviisha/darkvoid/internal/feature/feed/cache"
	feedhandler "github.com/jarviisha/darkvoid/internal/feature/feed/handler"
	feedservice "github.com/jarviisha/darkvoid/internal/feature/feed/service"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	"github.com/jarviisha/darkvoid/pkg/config"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// FeedContext represents the Feed bounded context with all its dependencies.
type FeedContext struct {
	// Services
	FeedService *feedservice.FeedService

	// Handlers
	FeedHandler *feedhandler.FeedHandler
}

// SetupFeedContext initializes the Feed context with all required dependencies.
// redisClient may be nil — in that case a no-op cache is used (pull on-the-fly).
// dikeCfg controls whether scoring is delegated to the Dike ranking service or uses the local formula.
func SetupFeedContext(store storage.Storage, postCtx *PostContext, userCtx *UserContext, followService *service.FollowService, redisClient *pkgredis.Client, dikeCfg config.DikeConfig) *FeedContext {
	ur := &userReader{userRepo: userCtx.UserRepo}
	pr := &postReader{
		postRepo:   postCtx.PostRepo,
		mediaRepo:  postCtx.MediaRepo,
		likeRepo:   postCtx.LikeRepo,
		userReader: ur,
	}

	fr := &followReader{followService: followService}
	lr := &likeReader{likeRepo: postCtx.LikeRepo}

	// Build ranker: Dike gRPC when enabled, local formula otherwise.
	var ranker feed.Ranker
	if dikeCfg.Enabled {
		ranker = setupDikeRanker(dikeCfg)
	}
	if ranker == nil {
		ranker = feed.NewLocalRanker(feed.DefaultScorerConfig())
	}

	// Build cache: Redis when available, no-op otherwise
	var fc feedcache.FeedCache
	if redisClient != nil {
		fc = feedcache.NewRedisFeedCache(redisClient)
	} else {
		fc = feedcache.NewNopFeedCache()
	}

	// Wire invalidator into FollowService so follow/unfollow clears the following IDs cache
	followService.WithFeedInvalidator(fc)
	// Wire trending invalidator into LikeService so like/unlike clears the trending cache
	postCtx.LikeService.WithTrendingInvalidator(fc)
	// Wire trending invalidator into CommentService so create/delete comment clears the trending cache
	postCtx.CommentService.WithTrendingInvalidator(fc)

	feedSvc := feedservice.NewFeedService(pr, fr, lr, ranker, fc)
	feedHdlr := feedhandler.NewFeedHandler(feedSvc, store)

	return &FeedContext{
		FeedService: feedSvc,
		FeedHandler: feedHdlr,
	}
}

// RegisterRoutes registers all routes for the Feed context.
func (ctx *FeedContext) RegisterRoutes(r chi.Router, auth middleware.AuthMiddleware) {
	// Discovery feed — public, enriched with is_liked when authenticated
	r.Group(func(r chi.Router) {
		r.Use(auth.Optional)
		r.Get("/discover", ctx.FeedHandler.GetDiscover)
	})

	// Personalized feed — authentication required
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)
		r.Get("/feed", ctx.FeedHandler.GetFeed)
	})
}
