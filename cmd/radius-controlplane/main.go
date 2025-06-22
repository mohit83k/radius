package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mohit83k/radius/internal/config"
	"github.com/mohit83k/radius/internal/logger"
	"github.com/mohit83k/radius/internal/redisclient"
	"github.com/mohit83k/radius/internal/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	cfg := config.Load()

	log, err := logger.NewLogrusLogger(cfg.LogFilePath)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	store := redisclient.NewRedisStore(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	addr := ":" + cfg.ServerPort

	radiusServer := server.NewServer(addr, cfg.RadiusSecret, store, log)

	if err := radiusServer.ListenAndServe(ctx); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
