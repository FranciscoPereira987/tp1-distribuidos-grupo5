package utils

import (
	"strings"

	"github.com/spf13/viper"
)

func InitConfig(prefix, configPath string) (*viper.Viper, error) {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvPrefix(prefix)

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath(configPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	return v, nil
}
