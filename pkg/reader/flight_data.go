package reader

import (
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	FLIGHT_TYPE_NUMBER = byte(0x09)
)

type FlightDataType struct {
	id          *typing.StrType
	origin      *typing.StrType
	destination *typing.StrType

	duration *typing.IntType
	fare     *typing.FloatType
	distance *typing.IntType

	stops *typing.StrType
}

func NewFlightDataType(id string,
	origin string, destination string,
	duration int, fare float64, distance int,
	stops string) (*FlightDataType, error) {
	idStr, err := typing.NewStr(id)
	if err != nil {
		return nil, err
	}
	originStr, err := typing.NewStr(origin)
	if err != nil {
		return nil, err
	}
	destinationStr, err := typing.NewStr(destination)
	if err != nil {
		return nil, err
	}
	durationInt := &typing.IntType{Value: uint32(duration)}
	fareFloat := &typing.FloatType{Value: fare}
	distanceInt := &typing.IntType{Value: uint32(distance)}
	stopsStr, err := typing.NewStr(stops)
	if err != nil {
		return nil, err
	}
	return &FlightDataType{
		id:          idStr,
		origin:      originStr,
		destination: destinationStr,
		duration:    durationInt,
		fare:        fareFloat,
		distance:    distanceInt,
		stops:       stopsStr,
	}, nil
}

func (flight *FlightDataType) Number() byte {
	return FLIGHT_TYPE_NUMBER
}

func (flight *FlightDataType) Serialize() []byte {
	header := utils.GetHeader(flight)
	header = append(header, flight.id.Serialize()...)
	header = append(header, flight.origin.Serialize()...)
	header = append(header, flight.destination.Serialize()...)
	header = append(header, flight.duration.Serialize()...)
	header = append(header, flight.fare.Serialize()...)
	header = append(header, flight.distance.Serialize()...)
	return append(header, flight.stops.Serialize()...)
}

func (flight *FlightDataType) Deserialize(stream []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("invalid stream")
		}

	}()
	if err = utils.CheckHeader(flight, stream); err != nil {
		return err
	}
	id, rest := typing.GetTypeFromStream(flight.id, stream[1:])
	if err = flight.id.Deserialize(id); err != nil {
		return err
	}

	origin, rest := typing.GetTypeFromStream(flight.origin, rest)
	if err = flight.origin.Deserialize(origin); err != nil {
		return err
	}
	destination, rest := typing.GetTypeFromStream(flight.destination, rest)
	if err = flight.destination.Deserialize(destination); err != nil {
		return err
	}
	duration, rest := typing.GetTypeFromStream(flight.duration, rest)
	if err = flight.duration.Deserialize(duration); err != nil {
		return err
	}
	fare, rest := typing.GetTypeFromStream(flight.fare, rest)
	if err = flight.fare.Deserialize(fare); err != nil {
		return err
	}
	distance, rest := typing.GetTypeFromStream(flight.distance, rest)
	if err = flight.distance.Deserialize(distance); err != nil {
		return err
	}
	stops, rest := typing.GetTypeFromStream(flight.stops, rest)
	if err = flight.stops.Deserialize(stops); err != nil {
		return err
	}
	if len(rest) > 0 {
		return errors.New("invalid flight data type")
	}
	return nil
}

func (flight *FlightDataType) Trim(stream []byte) (trimed []byte) {
	defer func() {
		if r := recover(); r != nil {
			trimed = nil
		}

	}()
	trimed = flight.id.Trim(stream[1:])
	trimed = flight.origin.Trim(trimed)
	trimed = flight.destination.Trim(trimed)
	trimed = flight.duration.Trim(trimed)
	trimed = flight.fare.Trim(trimed)
	trimed = flight.distance.Trim(trimed)
	trimed = flight.stops.Trim(trimed)
	return
}

func (flight *FlightDataType) AsRecord() []string {
	record := flight.id.AsRecord()
	record = append(record, flight.origin.AsRecord()...)
	record = append(record, flight.destination.AsRecord()...)
	record = append(record, flight.duration.AsRecord()...)
	record = append(record, flight.fare.AsRecord()...)
	record = append(record, flight.distance.AsRecord()...)
	return append(record, flight.stops.AsRecord()...)
}

func (flight *FlightDataType) IntoDistanceData() *distance.AirportDataType {
	data, _ := distance.NewAirportData(flight.id.Value(), flight.origin.Value(), flight.destination.Value(), flight.distance.Value)
	return data
}


func (flight *FlightDataType) IntoStopsFilterData() (data middleware.StopsFilterData) {
	data.ID = [16]byte([]byte(flight.id.Value()))
	data.Origin = flight.origin.Value()
	data.Destination = flight.destination.Value()
	data.Duration = flight.duration.Value
	data.Price = float32(flight.fare.Value)
	data.Stops = flight.stops.Value()
	return
}


func (flight *FlightDataType) IntoAvgFilterData() (data middleware.AvgFilterData) {
	data.Origin = flight.origin.Value()
	data.Destination = flight.destination.Value()
	data.Price = float32(flight.fare.Value)
	return
	
}