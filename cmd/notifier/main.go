package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/go-playground/validator/v10"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/api/handlers/notification"
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Must()
	val := validator.New()

	conn, err := rabbitmq.Connect("amqp://guest:guest@rabbitmq:5672", 3, 1)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to rabbitmq")
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to open channel")
	}
	defer ch.Close()

	q, err := queue.NewNotificationQueue(ch)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to create notification queue")
	}

	opts := &dbpg.Options{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	slaveDNSs := make([]string, 0, len(cfg.Database.Slaves))

	for _, s := range cfg.Database.Slaves {
		slaveDNSs = append(slaveDNSs, s.DSN())
	}

	db, err := dbpg.New(cfg.Database.Master.DSN(), slaveDNSs, opts)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Master.Close()

	repo := notifrepo.NewRepository(db)
	rdb := redis.New(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.Database)

	if err := rdb.Ping(ctx).Err(); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to redis")
	}

	emailClient := email.NewClient(
		cfg.Email.SMTPHost,
		cfg.Email.SMTPPort,
		cfg.Email.Username,
		cfg.Email.Password,
		cfg.Email.From,
	)
	telegramClient := telegram.NewClient(cfg.Telegram.Token)

	notifiers := map[string]notifsvc.Notifier{
		"email":    emailClient,
		"telegram": telegramClient,
	}

	service := notifsvc.NewService(repo, notifiers, rdb, q.Publisher)
	notifHandler := notification.NewHandler(service, val, cfg)
	messageHandler := notifmsg.NewHandler(service)

	notifier := worker.NewNotifier(q, messageHandler)

	go notifier.Run(ctx, cfg.Retry, cfg.Workers.Count)

	s := server.New(notifHandler)
	if err := s.Run(cfg.Server.HTTPPort); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to start server")
	}
}
