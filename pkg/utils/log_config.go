package utils

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func InitLogger(logLevel string) error {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	customFormatter := &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   false,
	}
	logrus.SetFormatter(customFormatter)
	logrus.SetLevel(level)
	return nil
}

func DefaultLogger() error {
	return InitLogger("DEBUG")
}

func PrintConfig(v *viper.Viper, configVars ...string) {
	var varsString string
	for _, variable := range configVars {
		varsString += variable + "=" + v.GetString(variable) + "; "
	}
	logrus.Infof("action: config | result: success | variables: %s", varsString)
}
