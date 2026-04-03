package app

import (
	"context"

	"github.com/jarviisha/darkvoid/internal/feature/feed"
	"github.com/jarviisha/darkvoid/pkg/config"
	"github.com/jarviisha/darkvoid/pkg/logger"
	dikeclient "github.com/jarviisha/dike/pkg/client"
	rankingv1 "github.com/jarviisha/dike/proto/ranking/v1"
)

// setupDikeRanker registers the project with Dike (if needed), creates the client,
// and returns a DikeRanker. Returns nil on any failure so the caller can fall back to local scoring.
func setupDikeRanker(cfg config.DikeConfig) feed.Ranker {
	ctx := context.Background()

	apiKey := cfg.APIKey
	if apiKey == "" {
		// Auto-register project to get an API key.
		var err error
		apiKey, err = dikeclient.RegisterProject(ctx, cfg.Addr, cfg.ProjectID, "DarkVoid feed ranking")
		if dikeclient.IsAlreadyExists(err) {
			logger.Error(ctx, "dike project already exists but no DIKE_API_KEY configured — set it in .env",
				"project", cfg.ProjectID)
			return nil
		}
		if err != nil {
			logger.Warn(ctx, "failed to register dike project, falling back to local scorer",
				"error", err, "addr", cfg.Addr)
			return nil
		}
		logger.Info(ctx, "dike project registered — save the API key to DIKE_API_KEY in .env",
			"project", cfg.ProjectID, "api_key", apiKey)
	}

	client, err := dikeclient.New(cfg.Addr, cfg.ProjectID, apiKey)
	if err != nil {
		logger.Warn(ctx, "failed to connect to dike ranking service, falling back to local scorer",
			"error", err, "addr", cfg.Addr)
		return nil
	}

	// Ensure default formula model exists — also serves as a connectivity check
	// since dikeclient.New uses lazy connection.
	if err := ensureDikeDefaultModel(ctx, client); err != nil {
		logger.Warn(ctx, "dike service unreachable, falling back to local scorer",
			"error", err, "addr", cfg.Addr)
		return nil
	}

	logger.Info(ctx, "dike ranking service connected", "addr", cfg.Addr, "project", cfg.ProjectID)
	return feed.NewDikeRanker(client, cfg.ModelID)
}

// ensureDikeDefaultModel upserts the default formula model matching the local scoring logic.
// UpsertFormula is idempotent — if the model already exists it bumps the version, which is harmless.
func ensureDikeDefaultModel(ctx context.Context, client *dikeclient.Client) error {
	components := []*rankingv1.FormulaComponent{
		{
			Name:     "engagement",
			Feature:  "engagement_count",
			Function: "log1p",
			Weight:   10.0,
		},
		{
			Name:     "recency",
			Feature:  "hours_age",
			Function: "decay",
			Weight:   1.0,
			Params:   map[string]float64{"scale": 20.0, "exponent": 1.5},
		},
		{
			Name:     "relationship",
			Feature:  "is_following",
			Function: "boolean",
			Weight:   1.0,
			Params:   map[string]float64{"bonus": 10.0},
		},
	}

	version, err := client.UpsertFormula(ctx, "feed_v1", true, components)
	if err != nil {
		return err
	}
	logger.Info(ctx, "dike default model ensured", "model", "feed_v1", "version", version)
	return nil
}
