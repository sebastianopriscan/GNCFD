package nvs

import (
	"errors"
	"fmt"
	"math"
)

/*
type SpacePoint interface{}

	type Space interface {
		Distance(*SpacePoint, *SpacePoint)
	}
*/
type NormedVectorSpace[SUPPORT float64 | complex128] struct {
	dimension   int
	distance    func([]SUPPORT, []SUPPORT) float64
	rescaling   func([]SUPPORT, float64) []SUPPORT
	externalMul func([]SUPPORT, float64) []SUPPORT
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

func (nvs *NormedVectorSpace[SUPPORT]) UnitVector(first *Point[SUPPORT], second *Point[SUPPORT]) (*Point[SUPPORT], error) {
	if nvs.dimension <= 0 || nvs.distance == nil {
		return nil, errors.New("dim should be greater than 0 and distance should not be nil")
	}
	if first.space != nvs || second.space != nvs {
		return nil, errors.New("the points do not belong to this space")
	}

	norm, err := nvs.Distance(first, second)
	if err != nil {
		return nil, fmt.Errorf("error in distance evaluation, details: %s", err)
	}

	toRescale := make([]SUPPORT, nvs.dimension)
	for i := 0; i < nvs.dimension; i++ {
		toRescale[i] = first.GetCoordinates()[i] - second.coordinates[i]
	}

	rescaled := nvs.rescaling(toRescale, norm)

	retPoint, err := NewPoint(nvs, rescaled)
	if err != nil {
		return nil, fmt.Errorf("error in new point creation, details: %s", err)
	}

	return retPoint, nil
}

func (nvs *NormedVectorSpace[SUPPORT]) ExternalMul(pt *Point[SUPPORT], val float64) (*Point[SUPPORT], error) {
	if nvs.dimension <= 0 || nvs.distance == nil {
		return nil, errors.New("dim should be greater than 0 and distance should not be nil")
	}
	if pt.space != nvs {
		return nil, errors.New("the point does not belong to this space")
	}

	newCoords := nvs.externalMul(pt.coordinates, val)

	return NewPoint(nvs, newCoords)
}

func NewNormedVectorSpace[SUPPORT float64 | complex128](dim int, distance func([]SUPPORT, []SUPPORT) float64,
	rescale func([]SUPPORT, float64) []SUPPORT, exmul func([]SUPPORT, float64) []SUPPORT) (*NormedVectorSpace[SUPPORT], error) {

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

func euclideanRescale(vector []float64, norm float64) []float64 {
	retVal := make([]float64, len(vector))
	for i, entry := range vector {
		retVal[i] = entry / norm
	}

	return retVal
}

func euclideanExMul(vector []float64, val float64) []float64 {

	retVal := make([]float64, len(vector))
	for i, entry := range vector {
		retVal[i] = entry * val
	}

	return retVal
}

func NewRealEuclideanSpace(dim int) (*NormedVectorSpace[float64], error) {
	return NewNormedVectorSpace(dim, euclideanNorm, euclideanRescale, euclideanExMul)
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
