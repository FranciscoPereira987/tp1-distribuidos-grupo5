package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/avgFilter/common"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
)

func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	// Configure viper to read env variables with the STOPS_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("avg")
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	v.SetDefault("source.kind", "direct")

	// Try to read configuration from config file. If config file
	// does not exists then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("cmd/stopsFilter")
	if err := v.ReadInConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "Configuration could not be read from config file. Using env variables instead")
	}

	if _, err := strconv.Atoi(v.GetString("id")); err != nil {
		return nil, fmt.Errorf("Could not parse AVG_ID env var as int: %w", err)
	}

	return v, nil
}

func InitLogger(logLevel string) error {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	customFormatter := &log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   false,
	}
	log.SetFormatter(customFormatter)
	log.SetLevel(level)
	return nil
}

// Describes the topology around this node.
func setupMiddleware(m *mid.Middleware, v *viper.Viper) (string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source.name"), v.GetString("source.kind"))
	if err != nil {
		return "", "", err
	}

	q, err := m.QueueDeclare(v.GetString("queue"))
	if err != nil {
		return "", "", err
	}

	// Subscribe to shards specific and EOF events.
	shardKey := mid.ShardKey(v.GetString("id"))
	err = m.QueueBind(q, source, []string{shardKey, "avg", "control"})
	if err != nil {
		return "", "", err
	}

	sink, err := m.QueueDeclare(v.GetString("results"))
	return q, sink, err
}

func main() {
	v, err := InitConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := InitLogger(v.GetString("log.level")); err != nil {
		log.Fatal(err)
	}

	middleware, err := mid.Dial(v.GetString("server.url"))
	if err != nil {
		log.Fatal(err)
	}
	defer middleware.Close()

	source, sink, err := setupMiddleware(middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	filter := common.NewFilter(middleware, source, sink)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	go func() {
		<-sig
		cancel(fmt.Errorf("Signal received"))
	}()

	if err := filter.Run(ctx); err != nil {
		log.Error(err)
	}
}
