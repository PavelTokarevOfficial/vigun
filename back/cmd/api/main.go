package main

import (
	"context"
	instagramclient "github.com/finde-clip/finde-v2/back/infrastructure/instagram"
	"github.com/finde-clip/finde-v2/back/infrastructure/postgres"
	"github.com/finde-clip/finde-v2/back/infrastructure/storage"
	"github.com/finde-clip/finde-v2/back/infrastructure/twitch"
	"github.com/finde-clip/finde-v2/back/internal/assets"
	"github.com/finde-clip/finde-v2/back/internal/clip"
	"github.com/finde-clip/finde-v2/back/internal/config"
	"github.com/finde-clip/finde-v2/back/internal/httpapi"
	"github.com/finde-clip/finde-v2/back/internal/instagram"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/finde-clip/finde-v2/back/internal/platform/db"
	"github.com/finde-clip/finde-v2/back/internal/processing"
	"github.com/finde-clip/finde-v2/back/internal/streamer"
	"github.com/finde-clip/finde-v2/back/internal/subscription"
	"github.com/finde-clip/finde-v2/back/internal/videotemplate"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, e := config.Load()
	if e != nil {
		log.Error("invalid config", "error", e)
		os.Exit(1)
	}
	ctx := context.Background()
	if e = db.Migrate(cfg.DatabaseURL, "migrations"); e != nil {
		log.Error("migration failed", "error", e)
		os.Exit(1)
	}
	pool, e := db.Open(ctx, cfg.DatabaseURL)
	if e != nil {
		log.Error("database unavailable", "error", e)
		os.Exit(1)
	}
	defer pool.Close()
	store, e := storage.New(ctx, storage.Settings{Endpoint: cfg.S3Endpoint, PublicEndpoint: cfg.S3PublicEndpoint, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, Bucket: cfg.S3Bucket, Region: cfg.S3Region, UseSSL: cfg.S3SSL})
	if e != nil {
		log.Error("storage unavailable", "error", e)
		os.Exit(1)
	}
	if e = store.EnsureBucket(ctx); e != nil {
		log.Error("bucket unavailable", "error", e)
		os.Exit(1)
	}
	assetService := assets.New(pool, store)
	templateService := videotemplate.New(pool, store)
	twitchClient := twitch.New(cfg.TwitchClientID, cfg.TwitchClientSecret)
	subscriptionService := subscription.New(postgres.NewSubscriptionRepository(pool), twitch.NewSubscriptionSource(twitchClient))
	instagramFactory := instagramclient.NewFactory(cfg.InstagramAPIVersion)
	instagramService := instagram.New(
		postgres.NewInstagramAccounts(pool),
		instagramFactory.New,
		instagramFactory,
		cfg.InstagramAppSecret,
	)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(streamer.New(pool), clip.New(pool, twitchClient, templateService), subscriptionService, assetService, templateService, media.NewLibrary(pool, store), media.NewVideos(pool, store), processing.NewJobs(pool), instagramService, log).Router(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("api started", "addr", cfg.HTTPAddr)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Error("server failed", "error", e)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(stop)
}
