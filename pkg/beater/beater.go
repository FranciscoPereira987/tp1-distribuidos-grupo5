package beater

import (
	"log"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func StartBeaterClient(v *viper.Viper) *BeaterClient {
	client, err := NewBeaterClient(v.GetString("name"), v.GetString("name")+":"+v.GetString("beater_port"))
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func StopBeaterClient(client *BeaterClient) {
	if err := client.Stop(); err != nil {
		logrus.Errorf("action: Stopping beater client | result: failed | reason: %s", err)
	}

}
