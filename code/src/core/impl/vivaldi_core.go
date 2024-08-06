package impl

import (
	"errors"
	"math"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/nvs"
)

type nodeData[SUPPORT float64 | complex128] struct {
	isFailed bool
	coords   *nvs.Point[SUPPORT]
}

type VivaldiCore[SUPPORT float64 | complex128] struct {
	nodesCache    map[int64]nodeData[SUPPORT]
	myGUID        int64
	myCoordinates *nvs.Point[SUPPORT]
	callback      func(rtt float64, metadata core.Metadata)
	space         *nvs.NormedVectorSpace[SUPPORT]
}

/*
	func (cr *VivaldiCore[T]) GetMyCoordinates() T {
		return cr.myCoordinates
	}

	func (cr *VivaldiCore[T]) GetCoordinatesOf(guid int64) T {
		return cr.nodesCache[guid].Coords
	}

	func (cr *VivaldiCore[T]) GetAllCoordinates() map[int64]T {
		retMap := make(map[int64]T)

		for k, v := range cr.nodesCache {

			retMap[k] = v.Coords
		}

		return retMap
	}
*/
func (cr *VivaldiCore[SUPPORT]) GetClosestOf(guids []int64) ([]int64, error) {
	min_distance := math.MaxFloat64
	var retSlice []int64

	for _, guid := range guids {
		point, ok := cr.nodesCache[guid]
		if !ok {
			continue
		}
		guid_distance, err := cr.space.Distance(cr.myCoordinates, point.coords)
		if err != nil {
			return make([]int64, 0), errors.New("the points whose distance was asked do not belong to the same space")
		}

		if min_distance > guid_distance {
			retSlice = append(make([]int64, 0), guid)
			min_distance = guid_distance
		} else if min_distance == guid_distance {
			retSlice = append(retSlice, guid)
		}
	}

	return retSlice, nil
}

func (cr *VivaldiCore[SUPPORT]) GetIsFailed(guid int64) bool {
	return cr.nodesCache[guid].isFailed
}

func (cr *VivaldiCore[T]) GetCallback() func(rtt float64, metadata core.Metadata) {
	return cr.callback
}

func NewVivaldiCore[SUPPORT float64 | complex128](myGuid int64, myCoords []SUPPORT, space *nvs.NormedVectorSpace[SUPPORT]) (*VivaldiCore[SUPPORT], error) {

	if space.Dimension() == 0 {
		return nil, errors.New("space malformed, please use the New* function to properly initialize one")
	}
	space_coords, err := nvs.NewPoint(space, myCoords)
	if err != nil {
		return nil, errors.New("initial coordinate not compatible with the requested space")
	}

	cr := &VivaldiCore[SUPPORT]{
		nodesCache:    make(map[int64]nodeData[SUPPORT]),
		myCoordinates: space_coords,
		myGUID:        myGuid,
		space:         space,
	}

	cr.callback = func(rtt float64, metadata core.Metadata) {
		//The Vivaldi algorithm
	}

	return cr, nil
}
