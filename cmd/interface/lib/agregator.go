package lib

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/sirupsen/logrus"
)

type AgregatorConfig struct {
	AgregatorQueue *protocol.Protocol
}

type Agregator struct {
	config        AgregatorConfig
	listeningChan chan *protocol.Protocol
	
}

func NewAgregator(config AgregatorConfig) *Agregator {
	return &Agregator{
		config:        config,
		listeningChan: make(chan *protocol.Protocol),
	}
}

func (agg *Agregator) GetChan() (chan<- *protocol.Protocol) {
	return agg.listeningChan
}




func (agg *Agregator) Run() error {
	defer func () {
		agg.config.AgregatorQueue.Close()
		agg.config.AgregatorQueue.Shutdown()
	}()
	results := <-agg.listeningChan
	
	data := getDataMessages()
	for {
		logrus.Info("Waiting for result data")
		if err := agg.config.AgregatorQueue.Recover(data); err != nil {
			logrus.Fatalf("error on aggregator queue: %s", err)
		}
		
		results.Send(data)
	}
	
	return nil
}
