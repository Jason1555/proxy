package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	stdhttp "net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	httpdelivery "proxy/internal/delivery/http"
	"proxy/internal/domain"
	"proxy/internal/infrastructure/config"
	ipaccessinfra "proxy/internal/infrastructure/ip_access"
	"proxy/internal/infrastructure/logger"
	"proxy/internal/infrastructure/metrics"
	pkgcache "proxy/internal/pkg/cache"
	"proxy/internal/repository"
	"proxy/internal/usecases"
)

const (
	defaultConfigPath   = "configs/config.example.yaml"
	defaultCacheEntries = 10000
	shutdownTimeout     = 10 * time.Second
)

// @title Proxy Service API
// @version 1.0
// @description Reverse proxy service
// @host localhost:8080
// @BasePath /proxy
func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config file (yaml|json|toml)")
	flag.Parse()

	bootstrapLogger, err := logger.NewZapLogger(logger.NewProdConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init bootstrap logger: %v\n", err)
		os.Exit(1)
	}
	var zlog logger.Logger = bootstrapLogger

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	manager := config.NewManager(*configPath, 0, config.NewFileLoader(), zlog)
	if err := manager.Load(ctx); err != nil {
		zlog.Fatalf("config load failed: %v", err)
	}

	cfg, ok := manager.Current()
	if !ok {
		zlog.Fatalf("config not loaded")
	}

	zlog = applyLogLevel(zlog, cfg.Log.Level)

	metrics.Init()

	monitoringSvc := usecases.NewMonitoringService(cfg.ToMonitoringConfig(), zlog)

	ipParser := ipaccessinfra.NewParser()
	ipMatcher := ipaccessinfra.NewMatcher(ipParser)
	ipCache := ipaccessinfra.NewSimpleIPFilterCache(cfg.IPAccess.CacheTTL.Duration())
	ipRepo := repository.NewInMemoryIPAccessRepository()
	if err := ipRepo.SavePolicy(ctx, cfg.ToIPAccessPolicy(time.Now())); err != nil {
		zlog.Fatalf("init ip policy: %v", err)
	}
	ipFilter := usecases.NewipFilterService(
		ipRepo, ipCache, ipParser, ipMatcher,
		zlog, cfg.ToIPFilterConfig(), monitoringSvc,
	)

	rlStore := repository.NewInMemoryRateLimitStore()
	rlSvc := usecases.NewRateLimitService(rlStore, cfg.ToRateLimitConfig(), zlog, monitoringSvc)

	cacheBackend := pkgcache.NewCache(repository.NewMapStorage(), domain.NewLRUPolicy(), defaultCacheEntries)
	cacheSvc := usecases.NewCacheService(cacheBackend, cfg.ToCacheConfig(), zlog)

	cacheHandler := httpdelivery.NewCacheHandler(cacheSvc)
	ipHandler := httpdelivery.NewIPFilterHandler(ipFilter)
	rlHandler := httpdelivery.NewRateLimitHandler(rlSvc)
	monHandler := httpdelivery.NewMonitoringHandler(monitoringSvc)

	ipMW := httpdelivery.NewIPFilterMiddleware(ipFilter)
	rlMW := httpdelivery.NewRateLimitMiddleware(rlSvc)
	cacheMW := httpdelivery.NewCacheMiddleware(cacheSvc)
	metricsMW := httpdelivery.MetricsMiddleware(monitoringSvc)

	reverseProxy, err := buildReverseProxy(cfg.Proxy.BaseURL, zlog, monitoringSvc)
	if err != nil {
		zlog.Fatalf("build reverse proxy: %v", err)
	}

	apiMux := stdhttp.NewServeMux()
	httpdelivery.NewRouter(cacheHandler, ipHandler, rlHandler, monHandler).Register(apiMux)

	root := stdhttp.NewServeMux()
	root.Handle("/metrics", promhttp.Handler())
	root.Handle("/api/", apiMux)

	proxyStack := metricsMW(
		ipMW.Middleware(
			rlMW.Handler(
				cacheMW.Middleware(reverseProxy),
			),
		),
	)
	root.Handle("/", proxyStack)

	server := &stdhttp.Server{
		Addr:         cfg.Server.Address,
		Handler:      root,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration(),
		WriteTimeout: cfg.Server.WriteTimeout.Duration(),
		IdleTimeout:  cfg.Server.IdleTimeout.Duration(),
	}

	unsubscribe := manager.Subscribe(func(change config.Change) {
		if change.Previous == nil {
			return
		}
		zlog.Infof("config changed, version=%d", change.Version)
		if err := ipFilter.SetPolicy(ctx, change.Current.ToIPAccessPolicy(time.Now())); err != nil {
			zlog.Errorf("apply new ip policy: %v", err)
		}
	})
	defer unsubscribe()

	watchCtx, cancelWatch := context.WithCancel(context.Background())
	defer cancelWatch()
	go runConfigWatcher(watchCtx, manager, cfg.ReloadInterval(), zlog)

	serverErr := make(chan error, 1)
	go func() {
		zlog.Infof("listening on %s, upstream=%s", cfg.Server.Address, cfg.Proxy.BaseURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		zlog.Infof("shutdown signal received")
	case err := <-serverErr:
		zlog.Errorf("server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		zlog.Errorf("graceful shutdown: %v", err)
	}
	zlog.Infof("server stopped")
}

func applyLogLevel(current logger.Logger, level string) logger.Logger {
	level = strings.TrimSpace(strings.ToLower(level))
	if level == "" {
		return current
	}

	cfg := logger.NewProdConfig()
	cfg.Level = level

	updated, err := logger.NewZapLogger(cfg)
	if err != nil {
		current.Warnf("invalid log level %q, keeping default: %v", level, err)
		return current
	}
	return updated
}

func buildReverseProxy(target string, log logger.Logger, mon domain.MonitoringCollector) (stdhttp.Handler, error) {
	upstream, err := url.Parse(strings.TrimSpace(target))
	if err != nil || upstream.Scheme == "" || upstream.Host == "" {
		return nil, fmt.Errorf("invalid upstream url %q", target)
	}

	rp := httputil.NewSingleHostReverseProxy(upstream)
	rp.ErrorHandler = func(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
		log.Errorf("upstream %s error: %v", upstream.Host, err)
		mon.RecordUpstream(domain.UpstreamMetric{
			Name:    upstream.Host,
			Healthy: false,
			Error:   err.Error(),
		})
		stdhttp.Error(w, "bad gateway", stdhttp.StatusBadGateway)
	}
	rp.ModifyResponse = func(resp *stdhttp.Response) error {
		mon.RecordUpstream(domain.UpstreamMetric{
			Name:       upstream.Host,
			Healthy:    resp.StatusCode < 500,
			StatusCode: resp.StatusCode,
		})
		return nil
	}
	return rp, nil
}

func runConfigWatcher(ctx context.Context, m *config.Manager, interval time.Duration, log logger.Logger) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.Load(ctx); err != nil {
				log.Warnf("config reload failed: %v", err)
			}
		}
	}
}
