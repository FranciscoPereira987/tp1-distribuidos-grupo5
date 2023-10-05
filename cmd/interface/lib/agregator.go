package lib

import (
	"io"

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

func (agg *Agregator) GetChan() chan<- *protocol.Protocol {
	return agg.listeningChan
}

func (agg *Agregator) Run() error {
	results := <-agg.listeningChan
	data := getDataMessages()
	for {
		if err := agg.config.AgregatorQueue.Recover(data); err != nil {
			if err.Error() == io.EOF.Error() {
				logrus.Info("agregator queue failed")
				break
			}
			continue
		}
		results.Send(data)
	}
	return nil
}
