package app

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/jarviisha/darkvoid/pkg/database"
)

// MetricsResponse represents the metrics response
type MetricsResponse struct {
	Service  ServiceMetrics  `json:"service"`
	Runtime  RuntimeMetrics  `json:"runtime"`
	Database DatabaseMetrics `json:"database"`
}

// ServiceMetrics represents service-level metrics
type ServiceMetrics struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Uptime      string `json:"uptime"`
}

// RuntimeMetrics represents Go runtime metrics
type RuntimeMetrics struct {
	GoVersion       string `json:"go_version"`
	NumGoroutines   int    `json:"num_goroutines"`
	NumCPU          int    `json:"num_cpu"`
	MemAllocMB      uint64 `json:"mem_alloc_mb"`
	MemTotalAllocMB uint64 `json:"mem_total_alloc_mb"`
	MemSysMB        uint64 `json:"mem_sys_mb"`
	NumGC           uint32 `json:"num_gc"`
}

// DatabaseMetrics represents database connection pool metrics
type DatabaseMetrics struct {
	MaxConns        int32  `json:"max_conns"`
	TotalConns      int32  `json:"total_conns"`
	IdleConns       int32  `json:"idle_conns"`
	AcquireCount    int64  `json:"acquire_count"`
	AcquireDuration string `json:"acquire_duration_avg"`
}

// metricsHandler handles metrics requests
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	response := s.collectMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("failed to encode metrics response", "error", err)
	}
}

// collectMetrics collects all application metrics
func (s *Server) collectMetrics() MetricsResponse {
	// Service metrics
	uptime := time.Since(s.startTime)
	serviceMetrics := ServiceMetrics{
		Name:        s.cfg.App.Name,
		Version:     s.cfg.App.Version,
		Environment: s.cfg.App.Environment,
		Uptime:      uptime.String(),
	}

	// Runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	runtimeMetrics := RuntimeMetrics{
		GoVersion:       runtime.Version(),
		NumGoroutines:   runtime.NumGoroutine(),
		NumCPU:          runtime.NumCPU(),
		MemAllocMB:      m.Alloc / 1024 / 1024,
		MemTotalAllocMB: m.TotalAlloc / 1024 / 1024,
		MemSysMB:        m.Sys / 1024 / 1024,
		NumGC:           m.NumGC,
	}

	// Database metrics
	var dbMetrics DatabaseMetrics
	if s.pool != nil {
		stats := database.GetStats(s.pool)
		dbMetrics = DatabaseMetrics{
			MaxConns:        stats.MaxConns,
			TotalConns:      stats.TotalConns,
			IdleConns:       stats.IdleConns,
			AcquireCount:    stats.AcquireCount,
			AcquireDuration: stats.AcquireDuration.String(),
		}
	}

	return MetricsResponse{
		Service:  serviceMetrics,
		Runtime:  runtimeMetrics,
		Database: dbMetrics,
	}
}
