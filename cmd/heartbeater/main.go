package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	close := make(chan os.Signal, 1)
	signal.Notify(close, syscall.SIGTERM)
	result := make(chan error, 1)
	for {
		invitation := invitation.Invitation(config)
		go func() {
			result <- invitation.Run()
		}()
		select {
		case err := <-result:
			logrus.Errorf("Invitation process ended with error: %s", err)
			logrus.Info("action: re-starting beater | status: in-progress")
		case <-close:
			logrus.Info("Action: recieved SIGTERM")
			invitation.Shutdown()
			logrus.Info("action: recieved SIGTERM | status: finished")
			return
		}
	}

}

func main() {
	invMain()
}
