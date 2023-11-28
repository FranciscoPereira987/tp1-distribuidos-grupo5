package main

import (
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/invitation"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func parseConfig(v *viper.Viper) (config *invitation.Config, err error) {
	port := v.GetString("net_port")
	id := v.GetUint("name")

	if config, err = invitation.NewConfigOn(id, port); err != nil {
		return config, err
	}
	peerPrefix := v.GetString("peers.prefix")
	for peerSuffix := v.GetUint("peers.count"); peerSuffix > 0; peerSuffix-- {
		peerNetName := fmt.Sprintf("%s%d", peerPrefix, peerSuffix)
		if id != peerSuffix {
			config.Mapping[peerSuffix] = peerNetName + ":" + port
			config.Peers = append(config.Peers, peerSuffix)
			config.Names = append(config.Names, peerNetName)
		} else {
			config.Name = peerNetName
		}
	}

	vContainers := v.Sub("containers")
	for _, worker := range []string{"demux", "distance", "fastest", "average"} {
		vWorker := vContainers.Sub(worker)
		workerPrefix := vWorker.GetString("prefix")
		for suffix := vWorker.GetUint("count"); suffix > 0; suffix-- {
			containerName := fmt.Sprintf("%s%d", workerPrefix, suffix)
			config.Names = append(config.Names, containerName)
		}
	}
	config.Heartbeat = v.GetString("heartbeat_port")

	return
}

func invMain() {
	v, err := utils.InitConfig("INV", "/config")
	if err != nil {
		logrus.Errorf("Error parsing config file: %s", err)
		return
	}

	config, err := parseConfig(v)
	if err != nil {
		logrus.Errorf("Error parsing config file: %s", err)
	}
	invitation := invitation.Invitation(config)
	if err := invitation.Run(); err != nil {
		logrus.Fatalf("Invitation process ended with error: %s", err)
	}
}

func main() {
	invMain()
}
