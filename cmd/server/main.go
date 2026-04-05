package main

import (
	"context"
	"log"

	"dispatch-socket-service/internal/auth"
	"dispatch-socket-service/internal/clients"
	"dispatch-socket-service/internal/config"
	httpserver "dispatch-socket-service/internal/http"
	"dispatch-socket-service/internal/http/handlers"
	"dispatch-socket-service/internal/services"
	"dispatch-socket-service/internal/utils"
	"dispatch-socket-service/internal/ws"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	logger := utils.NewLogger(cfg.LogLevel)
	rdbOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal(err)
	}
	rdb := redis.NewClient(rdbOpts)
	authenticator, err := auth.NewJWTAuthenticator(cfg.JWTSecret, cfg.JWTPublicKey)
	if err != nil {
		log.Fatal(err)
	}
	cm := ws.NewConnectionManager()
	presence := services.NewPresenceService(rdb, cfg.DriverStateTTL)
	location := services.NewLocationService(rdb, cfg.DriverStateTTL, cfg.DriverLocationTTL)
	coreClient := clients.NewCoreClient(cfg.CoreServiceBaseURL, cfg.CoreServiceTimeout)
	retrySync := services.NewRetrySyncService(rdb, coreClient, logger, cfg.CoreSyncRetryInterval, cfg.CoreSyncMaxRetries)
	go retrySync.Start(context.Background())
	coreSync := services.NewCoreSyncService(coreClient, retrySync, logger)
	offers := services.NewOfferDeliveryService(rdb, cm, cfg.WSWriteTimeout)
	accept := services.NewRideAcceptanceService(rdb, cm, offers, coreSync, cfg.WSWriteTimeout, logger)
	router := ws.NewMessageRouter(presence, location, accept, logger)
	wsHandler := ws.NewDriverWSHandler(authenticator, cm, presence, router, cfg.WSPingInterval, cfg.WSReadTimeout, cfg.WSWriteTimeout, logger)

	healthHandler := handlers.NewHealthHandler()
	dispatchHandler := handlers.NewInternalDispatchHandler(offers)
	server := httpserver.NewRouter(healthHandler, dispatchHandler, wsHandler)
	if err := server.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
