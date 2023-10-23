package distance

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
)

var ErrNotFound = errors.New("unknown airport code")

const cacheSize = 32
const earthRadiusMi = 3958

type radCoords struct {
	Lat, Lon float64
}

type distance struct {
	a, b string
	dist float64
}

type DistanceComputer struct {
	coordinates map[string]radCoords
	cache       [cacheSize]distance
}

func NewComputer() *DistanceComputer {
	return &DistanceComputer{
		coordinates: make(map[string]radCoords),
	}
}

func degreesToRadians(deg float64) float64 {
	return math.Pi / 180 * deg
}

func (comp *DistanceComputer) AddAirportCoords(code string, lat, lon float64) {
	comp.coordinates[code] = radCoords{
		Lat: degreesToRadians(lat),
		Lon: degreesToRadians(lon),
	}
}

func (comp *DistanceComputer) Distance(a, b string) (float64, error) {
	if distance, ok := comp.CachedDistance(a, b); ok {
		return distance, nil
	}

	p, ok := comp.coordinates[a]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, a)
	}
	q, ok := comp.coordinates[b]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, b)
	}

	// https://en.wikipedia.org/wiki/Spherical_law_of_cosines
	c1 := math.Cos(p.Lat - q.Lat)
	c2 := math.Cos(p.Lat + q.Lat)
	c3 := math.Cos(p.Lon - q.Lon)

	c := (1+c1)/2 - (1-c3)*(c1+c2)/4
	// clamp to account for floating point error
	c = max(0, min(1, c))
	dUnit := 2 * math.Acos(math.Sqrt(c))

	distance := dUnit * earthRadiusMi
	comp.CacheStore(a, b, distance)

	return distance, nil
}

func cacheKey(a, b string) uint32 {
	h1 := fnv.New32()
	h2 := fnv.New32()

	h1.Write([]byte(a))
	h2.Write([]byte(b))

	return (h1.Sum32() + h2.Sum32()) % cacheSize
}

func (comp *DistanceComputer) CacheStore(a, b string, dist float64) {
	comp.cache[cacheKey(a, b)] = distance{a, b, dist}
}

func (comp DistanceComputer) CachedDistance(a, b string) (float64, bool) {
	v := comp.cache[cacheKey(a, b)]
	return v.dist, (v.a == a && v.b == b) || (v.a == b && v.b == a)
}
