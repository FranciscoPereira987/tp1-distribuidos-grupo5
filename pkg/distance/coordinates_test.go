package distance_test

import (
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
)

func TestCoordinatesSerializationAndDeserialization(t *testing.T) {
	coords := distance.Coordinates{
		Lat: 10.1,
		Lon: 12.2,
	}

	data := distance.IntoData(coords, "AEX")
	Marshaled := data.Marshal()
	resultCoords := &distance.Coordinates{
		Lat: 0,
		Lon: 0,
	}
	otherData := distance.IntoData(*resultCoords, "")

	if err := otherData.UnMarshal(Marshaled); err != nil {
		t.Fatalf("error on unMarshaling: %s", err)
	}

	resultCoords, _ = distance.CoordsFromData(otherData)

	if (*resultCoords) != coords {
		t.Fatalf("expected: %f, %f\ngot: %f, %f", resultCoords.Lat, resultCoords.Lon, coords.Lat, coords.Lon)
	}

}
