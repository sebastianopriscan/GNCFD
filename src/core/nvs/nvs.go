package nvs

import (
	"errors"
	"math"
)

/*
type SpacePoint interface{}

	type Space interface {
		Distance(*SpacePoint, *SpacePoint)
	}
*/
type NormedVectorSpace[SUPPORT float64 | complex128] struct {
	dimension int
	distance  func([]SUPPORT, []SUPPORT) float64
}

func (nvs *NormedVectorSpace[SUPPORT]) Distance(first *Point[SUPPORT], second *Point[SUPPORT]) (float64, error) {
	if nvs.dimension <= 0 || nvs.distance == nil {
		return -1., errors.New("dim should be greater than 0 and distance should not be nil")
	}
	if first.space != nvs || second.space != nvs {
		return -1., errors.New("the points do not belong to this space")
	}

	return nvs.distance(first.coordinates, second.coordinates), nil
}

func (nvs *NormedVectorSpace[SUPPORT]) Dimension() int {
	return nvs.dimension
}

func NewNormedVectorSpace[SUPPORT float64 | complex128](dim int, distance func([]SUPPORT, []SUPPORT) float64) (*NormedVectorSpace[SUPPORT], error) {
	if dim <= 0 || distance == nil {
		return nil, errors.New("dim should be greater than 0 and distance should be nil")
	}
	return &NormedVectorSpace[SUPPORT]{}, nil
}

func euclideanNorm(first []float64, second []float64) float64 {
	sum := 0.
	for i := 0; i < len(first); i++ {
		sum += math.Pow(first[i]-second[i], 2.)
	}

	return math.Sqrt(sum)
}

func NewRealEuclideanSpace(dim int) (*NormedVectorSpace[float64], error) {
	return NewNormedVectorSpace(dim, euclideanNorm)
}

type Point[SUPPORT float64 | complex128] struct {
	space       *NormedVectorSpace[SUPPORT]
	coordinates []SUPPORT
}

func (pt *Point[SUPPORT]) GetCoordinates() []SUPPORT {
	return pt.coordinates
}

func (pt *Point[SUPPORT]) SetCoordinates(coords []SUPPORT) bool {
	if len(coords) != pt.space.dimension {
		return false
	}

	pt.coordinates = coords
	return true
}

func NewPoint[SUPPORT float64 | complex128](space *NormedVectorSpace[SUPPORT], coords []SUPPORT) (*Point[SUPPORT], error) {
	if len(coords) != space.dimension {
		return nil, errors.New("the point is incompatible with the requested space")
	}

	return &Point[SUPPORT]{space: space, coordinates: coords}, nil
}
