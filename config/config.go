package config

import (
	"github.com/spf13/viper"
)

var Cfg = viper.New()

func Parse(file string) error {
	Cfg.SetConfigType("yaml")
	Cfg.SetConfigFile(file)
	return Cfg.ReadInConfig()
}
