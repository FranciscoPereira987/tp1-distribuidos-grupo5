package beater

import (
	"log"

	"github.com/spf13/viper"
)


func StartBeaterClient(v *viper.Viper) *BeaterClient {
	client, err := NewBeaterClient(v.GetString("name"), v.GetString("name")+":"+v.GetString("beater_port"))
	if err != nil {
		log.Fatal(err)
	}
	return client
}