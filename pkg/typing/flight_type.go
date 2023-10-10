package typing

import (
	"bytes"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	DURATION_EXP = "[0-9]+"
)

type FlightDataType struct {
	Id [16]byte
	Origin string
	Destination string
	Duration string
	Fare string
	Distance string

	Stops string
}

func NewFlightData() *FlightDataType{
	return new(FlightDataType)
}

func (flight *FlightDataType) Number() byte {
	return middleware.FlightType
}

func (flight *FlightDataType) AsRecord() []string{
	record := []string{hex.EncodeToString(flight.Id[:])}

	return append(record, flight.Origin, flight.Destination, flight.Duration,
	flight.Fare, flight.Distance, flight.Stops)
}

func (flight *FlightDataType) Serialize() []byte {
	buf := make([]byte, 1)
	buf[0] = middleware.FlightType

	buf = append(buf, flight.Id[:]...)
	buf = middleware.AppendString(buf, flight.Origin)
	buf = middleware.AppendString(buf, flight.Destination)
	buf = middleware.AppendString(buf, flight.Duration)
	buf = middleware.AppendString(buf, flight.Fare)
	buf = middleware.AppendString(buf, flight.Distance)

	return middleware.AppendString(buf, flight.Stops)
}

func (flight *FlightDataType) Deserialize(r []byte)  error {
	if err := utils.CheckHeader(flight, r); err != nil {
		return err
	}
	if len(r) < 17 {
		return errors.New("invalid stream")
	}
	var err error
	flight.Id = [16]byte(r[1:17])
	reader := bytes.NewReader(r[17:])
	flight.Origin, err = middleware.ReadString(reader)

	if err == nil {
		flight.Destination, err = middleware.ReadString(reader)
	}

	if err == nil {
		flight.Duration, err = middleware.ReadString(reader)
	}

	if err == nil {
		flight.Fare, err = middleware.ReadString(reader)
	}

	if err == nil {
		flight.Distance, err = middleware.ReadString(reader)
	}

	if err == nil {
		flight.Stops, err = middleware.ReadString(reader)
	}

	return err
}

func (flight *FlightDataType) IntoStopsFilterData() (data middleware.StopsFilterData) {
	data.ID = flight.Id
	data.Origin = flight.Origin
	data.Destination = flight.Destination
	price, _ := strconv.ParseFloat(flight.Fare, 32)
	data.Price = float32(price)
	data.Stops = flight.Stops
	duration, _ := ParseDuration(flight.Duration)
	data.Duration = uint32(duration)
	return
}

func ParseDuration(duration string) (int, error) {
	exp, err := regexp.Compile(DURATION_EXP)
	if err != nil {
		return 0, err
	}
	values := exp.FindAllString(duration, 2)
	if len(values) < 1 {
		return 0, errors.New("invalid duration string")
	}
	if len(values) < 2 {
		values = append(values, "0")
	}
	hours, _ := strconv.Atoi(values[0])
	minutes, _ := strconv.Atoi(values[1])

	return minutes + hours*60, nil
}

func (flight *FlightDataType) IntoDistanceData() (data middleware.DataQ2) {
	data.ID = flight.Id
	data.Origin = flight.Origin
	data.Destination = flight.Destination
	distance, _ := strconv.ParseFloat(flight.Distance, 32)
	data.TotalDistance = uint32(distance)


	return
}

func (flight *FlightDataType) IntoAvgFilterData() (data middleware.AvgFilterData) {
	data.Origin = flight.Origin
	data.Destination = flight.Destination
	price, _ := strconv.ParseFloat(flight.Fare, 32)
	data.Price = float32(price)
	return
}