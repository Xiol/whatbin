package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Init() error {
	viper.SetConfigName("whatbin")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/whatbin")
	viper.AddConfigPath("$HOME/.config/whatbin/whatbin")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("WHATBIN")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config init error: %s", err)
	}
	return nil
}
