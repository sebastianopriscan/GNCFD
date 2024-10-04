package nvs

import (
	"errors"
	"fmt"
)

type NVSFunctions[SUPPORT float64 | complex128] struct {
	Distance    func([]SUPPORT, []SUPPORT) float64
	Rescaling   func([]SUPPORT, float64) []SUPPORT
	ExternalMul func([]SUPPORT, float64) []SUPPORT
	RandomEl    func() SUPPORT
	Zero        func(int) []SUPPORT
}

type NormedVectorSpace[SUPPORT float64 | complex128] struct {
	dimension   int
	distance    func([]SUPPORT, []SUPPORT) float64
	rescaling   func([]SUPPORT, float64) []SUPPORT
	externalMul func([]SUPPORT, float64) []SUPPORT
	randomEl    func() SUPPORT
	zero        func(int) []SUPPORT
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

	var rescaled []SUPPORT

	if norm != 0. {
		toRescale := make([]SUPPORT, nvs.dimension)
		for i := 0; i < nvs.dimension; i++ {
			toRescale[i] = first.GetCoordinates()[i] - second.coordinates[i]
		}

		rescaled = nvs.rescaling(toRescale, norm)
	} else {
		rescaled = make([]SUPPORT, nvs.dimension)
	RESCALED:
		for i := 0; i < nvs.dimension; i++ {
			rescaled[i] = nvs.randomEl()
		}
		zero := nvs.zero(nvs.dimension)
		norm := nvs.distance(rescaled, zero)
		if norm == 0 {
			goto RESCALED
		}
		rescaled = nvs.rescaling(rescaled, norm)
	}

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

func NewNormedVectorSpace[SUPPORT float64 | complex128](dim int, ops *NVSFunctions[SUPPORT]) (*NormedVectorSpace[SUPPORT], error) {

	if dim <= 0 || ops.Distance == nil || ops.ExternalMul == nil || ops.RandomEl == nil || ops.Rescaling == nil || ops.Zero == nil {
		return nil, errors.New("dim should be greater than 0 and no opration should be nil")
	}
	return &NormedVectorSpace[SUPPORT]{
		dimension:   dim,
		distance:    ops.Distance,
		rescaling:   ops.Rescaling,
		externalMul: ops.ExternalMul,
		randomEl:    ops.RandomEl,
		zero:        ops.Zero,
	}, nil
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
