package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
)

type Config struct {
	HTTPPort string         `mapstructure:"http_port"`
	Database Database       `mapstructure:"database"`
	Retry    retry.Strategy `mapstructure:"retry"`
}

type Database struct {
	Host    string
	Port    string
	User    string
	Pass    string
	Name    string
	SSLMode string `mapstructure:"ssl_mode"`
}

// DatabaseURL builds a PostgreSQL connection string
// based on the Database configuration.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Database.User,
		c.Database.Pass,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func Must() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		zlog.Logger.Panic().Err(err).Msg("failed to read config")
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		zlog.Logger.Panic().Err(err).Msg("failed to unmarshal config")
	}

	cfg.Database.Host = os.Getenv("DB_HOST")
	cfg.Database.Port = os.Getenv("DB_PORT")
	cfg.Database.User = os.Getenv("DB_USER")
	cfg.Database.Pass = os.Getenv("DB_PASSWORD")
	cfg.Database.Name = os.Getenv("DB_NAME")

	return &cfg
}
