package lib

import (
	"context"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

type AgregatorConfig struct {
	AgregatorQueue string
	Mid            *middleware.Middleware
	Ctx            context.Context
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
	defer agg.config.Mid.Close()
	results := <-agg.listeningChan
	defer results.Close()
	ch, err := agg.config.Mid.ConsumeWithContext(agg.config.Ctx, agg.config.AgregatorQueue)
	if err != nil {
		return err
	}
	for {
		data, more := <-ch
		result, err := typing.ResultsUnmarshal(data)
		if err == nil {
			logrus.Info("action: sending results | action: sending new result")
			data := protocol.NewDataMessage(result)
			results.Send(data)
		}
		if !more {
			logrus.Info("action: sending results | status: finished")
			break
		}
		
	}

	return nil
}
