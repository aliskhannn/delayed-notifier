// Package main initializes and runs the delayed-notifier service.
//
// It sets up connections to RabbitMQ, PostgreSQL, and Redis, configures
// email and telegram notifiers, starts the HTTP server, and launches
// background workers to process notifications from the queue.
package main

import (
	"context"
	"errors"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/api/handlers/notification"
	"github.com/aliskhannn/delayed-notifier/internal/api/router"
	"github.com/aliskhannn/delayed-notifier/internal/api/server"
	"github.com/aliskhannn/delayed-notifier/internal/config"
	notifmsg "github.com/aliskhannn/delayed-notifier/internal/rabbitmq/handlers/notification"
	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
	notifrepo "github.com/aliskhannn/delayed-notifier/internal/repository/notification"
	notifsvc "github.com/aliskhannn/delayed-notifier/internal/service/notification"
	"github.com/aliskhannn/delayed-notifier/internal/worker"
	"github.com/aliskhannn/delayed-notifier/pkg/email"
	"github.com/aliskhannn/delayed-notifier/pkg/telegram"
)

func main() {
	// Setup context to handle SIGINT and SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize logger and configuration.
	zlog.Init()
	cfg := config.Must()
	val := validator.New()

	// Connect to RabbitMQ.
	conn, err := rabbitmq.Connect(cfg.RabbitMQ.URL(), cfg.RabbitMQ.Retries, cfg.RabbitMQ.Pause)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to rabbitmq")
	}

	ch, err := conn.Channel()
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to open channel")
	}

	// Create notification queue.
	q, err := queue.NewNotificationQueue(ch, cfg)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to create notification queue")
	}

	// Connect to PostgreSQL master and slave databases.
	opts := &dbpg.Options{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	slaveDNSs := make([]string, 0, len(cfg.Database.Slaves))

	for _, s := range cfg.Database.Slaves {
		slaveDNSs = append(slaveDNSs, s.DSN())
	}
	zlog.Logger.Info().Msgf("db url: %s", cfg.Database.Master.DSN())
	db, err := dbpg.New(cfg.Database.Master.DSN(), slaveDNSs, opts)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to database")
	}

	// Connect to Redis
	dbNum, err := strconv.Atoi(cfg.Redis.Database)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to parse redis database")
	}

	zlog.Logger.Info().Msgf("redis config: %s, %s, %d", cfg.Redis.Address, cfg.Redis.Password, dbNum)
	rdb := redis.New(cfg.Redis.Address, cfg.Redis.Password, dbNum)

	if err = rdb.Ping(ctx).Err(); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to redis")
	}

	// Initialize email and telegram clients.
	smtpPort, err := strconv.Atoi(cfg.Email.SMTPPort)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to parse email smtp port")
	}

	emailClient := email.NewClient(
		cfg.Email.SMTPHost,
		smtpPort,
		cfg.Email.Username,
		cfg.Email.Password,
		cfg.Email.From,
	)
	telegramClient := telegram.NewClient(cfg.Telegram.Token)

	notifiers := map[string]notifsvc.Notifier{
		"email":    emailClient,
		"telegram": telegramClient,
	}

	// Initialize notification repository, service and handlers.
	repo := notifrepo.NewRepository(db)
	service := notifsvc.NewService(repo, q, notifiers, rdb)
	notifHandler := notification.NewHandler(service, val, cfg)
	messageHandler := notifmsg.NewHandler(service)

	// Start background notifier worker.
	notifier := worker.NewNotifier(q, messageHandler, service)
	go notifier.Run(ctx, cfg.Retry, cfg.Workers.Count)

	// Start HTTP server
	r := router.New(notifHandler)
	s := server.New(cfg.Server.HTTPPort, r)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			zlog.Logger.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for shutdown signal.
	<-ctx.Done()
	zlog.Logger.Info().Msg("shutdown signal received")

	// Graceful shutdown with timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	zlog.Logger.Info().Msg("shutting down server")
	if err := s.Shutdown(shutdownCtx); err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to shutdown server")
	}
	if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
		zlog.Logger.Info().Msg("timeout exceeded, forcing shutdown")
	}

	// Close master and slave databases.
	if err := db.Master.Close(); err != nil {
		zlog.Logger.Printf("failed to close master DB: %v", err)
	}
	for i, s := range db.Slaves {
		if err := s.Close(); err != nil {
			zlog.Logger.Printf("failed to close slave DB %d: %v", i, err)
		}
	}

	// Close RabbitMQ channel and connection
	if err := ch.Close(); err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to close RabbitMQ channel")
	}
	if err := conn.Close(); err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to close RabbitMQ connection")
	}
}
