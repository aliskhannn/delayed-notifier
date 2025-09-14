package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
)

type Config struct {
	Server   Server         `mapstructure:"server"`
	Database Database       `mapstructure:"database"`
	RabbitMQ RabbitMQ       `mapstructure:"rabbitmq"`
	Redis    Redis          `mapstructure:"redis"`
	Email    Email          `mapstructure:"email"`
	Telegram Telegram       `mapstructure:"telegram"`
	Retry    retry.Strategy `mapstructure:"retry"`
	Workers  struct {
		Count int `mapstructure:"count"`
	}
}

type Server struct {
	HTTPPort string `mapstructure:"http_port"`
}

type Database struct {
	Master DatabaseNode   `mapstructure:"master"`
	Slaves []DatabaseNode `mapstructure:"slaves"`

	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type DatabaseNode struct {
	Host    string `mapstructure:"host"`
	Port    string `mapstructure:"port"`
	User    string `mapstructure:"user"`
	Pass    string `mapstructure:"pass"`
	Name    string `mapstructure:"name"`
	SSLMode string `mapstructure:"ssl_mode"`
}

type RabbitMQ struct {
	Host       string        `mapstructure:"host"`
	Port       int           `mapstructure:"port"`
	User       string        `mapstructure:"user"`
	Password   string        `mapstructure:"password"`
	Retries    int           `mapstructure:"retries"`
	Pause      time.Duration `mapstructure:"pause"`
	Exchange   string        `mapstructure:"exchange"`
	Queue      string        `mapstructure:"queue"`
	RetryQueue string        `mapstructure:"retry_queue"`
	DLQ        string        `mapstructure:"dlq"`
	RoutingKey string        `mapstructure:"routing_key"`
}

type Redis struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type Email struct {
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort string `mapstructure:"smtp_port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type Telegram struct {
	Token  string `mapstructure:"token"`
	ChatID string `mapstructure:"chat_id"`
}

func (r RabbitMQ) URL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d",
		r.User, r.Password, r.Host, r.Port,
	)
}

func (n DatabaseNode) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		n.User, n.Pass, n.Host, n.Port, n.Name, n.SSLMode,
	)
}

func mustBindEnv() {
	bindings := map[string]string{
		"database.master.host": "DB_HOST",
		"database.master.port": "DB_PORT",
		"database.master.user": "DB_USER",
		"database.master.pass": "DB_PASSWORD",
		"database.master.name": "DB_NAME",

		"redis.address":  "REDIS_ADDRESS",
		"redis.password": "REDIS_PASSWORD",
		"redis.database": "REDIS_DATABASE",

		"email.smtp_host": "SMTP_HOST",
		"email.smtp_port": "SMTP_PORT",
		"email.username":  "SMTP_USER",
		"email.password":  "SMTP_PASS",
		"email.from":      "SMTP_FROM",

		"telegram.token":   "TELEGRAM_TOKEN",
		"telegram.chat_id": "TELEGRAM_CHAT_ID",

		"rabbitmq.host":     "RABBITMQ_HOST",
		"rabbitmq.port":     "RABBITMQ_PORT",
		"rabbitmq.user":     "RABBITMQ_USER",
		"rabbitmq.password": "RABBITMQ_PASSWORD",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			zlog.Logger.Panic().Err(err).Msgf("failed to bind env %s", env)
		}
	}
}

func Must() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		zlog.Logger.Panic().Err(err).Msg("failed to read config")
	}

	mustBindEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		zlog.Logger.Panic().Err(err).Msgf("failed to unmarshal config: %v", err)
	}

	return &cfg
}
