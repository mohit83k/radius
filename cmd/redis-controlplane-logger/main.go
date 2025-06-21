package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mohit83k/radius/internal/config"
	"github.com/mohit83k/radius/internal/logger"
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
		panic("failed to init logger: " + err.Error())
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            cfg.RedisAddr,
		Password:        cfg.RedisPass,
		DB:              cfg.RedisDB,
		MaxRetries:      5,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 1 * time.Second,
	})

	pubsub := rdb.PSubscribe(ctx, "__keyevent@*__:set")
	log.Info("Started Redis subscriber for SET events")

	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down Redis subscriber")
			_ = pubsub.Close()
			return

		case msg := <-pubsub.Channel():
			if !strings.HasPrefix(msg.Payload, "radius:acct:") {
				continue
			}

			timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
			log.WithFields(map[string]any{
				"timestamp": timestamp,
				"key":       msg.Payload,
			}).Info("Received update for RADIUS accounting key")
		}
	}
}
