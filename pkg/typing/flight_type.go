package typing

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var DurationRegexp = regexp.MustCompile(`P(\d+D)?T(\d+H)?(\d+M)?`)
var ErrInvalidDuration = errors.New("invalid duration format")

type FlightDataType struct {
	Id          [16]byte
	Origin      string
	Destination string
	Duration    string
	Fare        string
	Distance    string

	Stops string
}

func NewFlightData() *FlightDataType {
	return new(FlightDataType)
}

func (flight *FlightDataType) Number() byte {
	return middleware.FlightType
}

func (flight *FlightDataType) AsRecord() []string {
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

func (flight *FlightDataType) Deserialize(r []byte) error {
	if err := utils.CheckHeader(flight, r); err != nil {
		return err
	}
	if len(r) < 17 {
		return io.ErrUnexpectedEOF
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

func (flight *FlightDataType) IntoResultQ1() (data middleware.ResultQ1) {
	data.ID = flight.Id
	data.Origin = flight.Origin
	data.Destination = flight.Destination
	price, _ := strconv.ParseFloat(flight.Fare, 32)
	data.Price = float32(price)
	data.Stops = flight.Stops
	return
}

func ParseDuration(duration string) (minutes int, err error) {
	values := DurationRegexp.FindStringSubmatch(duration)
	if len(values) == 0 {
		return 0, fmt.Errorf("%w: '%s'", ErrInvalidDuration, duration)
	}

	days, hours, minutes := 0, 0, 0
	if values[1] != "" {
		days, err = strconv.Atoi(values[1][:len(values[1]-1)])
	}
	if err == nil && values[2] != "" {
		hours, err = strconv.Atoi(values[2][:len(values[2]-1)])
	}
	if err == nil && values[3] != "" {
		minutes, err = strconv.Atoi(values[3][:len(values[3]-1)])
	}

	return days*24*60 + hours*60 + minutes, err
}

func (flight *FlightDataType) IntoFastestFilterData() (data middleware.FastestFilterData) {
	data.ID = flight.Id
	data.Origin = flight.Origin
	data.Destination = flight.Destination
	data.Stops = flight.Stops
	duration, _ := ParseDuration(flight.Duration)
	data.Duration = uint32(duration)
	return
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
