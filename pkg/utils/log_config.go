package utils

import (
	"fmt"
	"strings"

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
	var b strings.Builder
	for i, variable := range configVars {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s=%s", variable, v.GetString(variable))
	}
	logrus.Debug("action: config | result: success | variables: ", b.String())
}
