package utils

import (
	"strings"

	"github.com/spf13/viper"
)

func InitConfig(configPath string, configVars ...string) (*viper.Viper, error) {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvPrefix("cli")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, envVar := range configVars {
		newVar := strings.Split(envVar, ".")
		v.BindEnv(newVar...)
	}

	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	return v, nil
}
