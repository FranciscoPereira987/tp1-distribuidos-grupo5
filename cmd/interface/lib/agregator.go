package lib

import (
	"context"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
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
	defer func() {
		agg.config.Mid.Close()
	}()
	results := <-agg.listeningChan
	defer func() {
		results.Close()
	}()
	ch, err := agg.config.Mid.ConsumeWithContext(agg.config.Ctx, agg.config.AgregatorQueue)
	if err != nil {
		return err
	}
	for {
		data, more := <-ch
		if !more {
			logrus.Info("action: sending results | status: finished")
			break
		}
		_, result, err := middleware.DistanceFilterUnmarshal(data)
		if err == nil {
			value := result.(middleware.DataQ2)
			dataType, _ := distance.NewAirportData(string(value.ID[:]), value.Origin, value.Destination, value.TotalDistance)
			data := protocol.NewDataMessage(dataType)
			results.Send(data)
		}
	}

	return nil
}
