package main

import (
	wbfconfig "github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/config"
)

func main() {
	cfgLoader := wbfconfig.New()

	if err := cfgLoader.Load("config.yml"); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load config")
	}

	var cfg config.Config
	if err := cfgLoader.Unmarshal(&cfg); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to unmarshal config")
	}
}
