package distance

import (
	"errors"
	"fmt"

	"github.com/umahmood/haversine"
)




type DistanceComputer struct {
	coordinates map[string]Coordinates
}

func NewComputer() *DistanceComputer {
	return &DistanceComputer{
		make(map[string]Coordinates),
	}
}

func (comp *DistanceComputer) AddAirport(name string, coordinates Coordinates) {
	comp.coordinates[name] = coordinates
}

func (comp DistanceComputer) CalculateDistance(origin string, destination string) (float64, error) {
	originCoor, okOrigin := comp.coordinates[origin]
	destinationCoor, okDest := comp.coordinates[destination]
	var err error
	if !okOrigin {
		err = errors.New(fmt.Sprintf("invalid name for airport: %s", origin))
	}
	if !okDest {
		err = errors.New(fmt.Sprintf("invalid name for airport: %s", destination))
	}
	if err != nil {
		return 0, err
	}
	_, distance := haversine.Distance(originCoor, destinationCoor)
	return distance, nil
}