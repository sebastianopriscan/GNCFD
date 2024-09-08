package vivaldi

import (
	"errors"
	"math"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/core/nvs"
)

type NodeData[SUPPORT float64 | complex128] struct {
	IsFailed bool
	Coords   *nvs.Point[SUPPORT]
}

type VivaldiCore[SUPPORT float64 | complex128] struct {
	nodesCache    map[guid.Guid]NodeData[SUPPORT]
	myGUID        guid.Guid
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
func (cr *VivaldiCore[SUPPORT]) GetClosestOf(guids []guid.Guid) ([]guid.Guid, error) {
	min_distance := math.MaxFloat64
	var retSlice []guid.Guid

	for _, single_guid := range guids {
		point, ok := cr.nodesCache[single_guid]
		if !ok {
			continue
		}
		guid_distance, err := cr.space.Distance(cr.myCoordinates, point.Coords)
		if err != nil {
			return make([]guid.Guid, 0), errors.New("the points whose distance was asked do not belong to the same space")
		}

		if min_distance > guid_distance {
			retSlice = append(make([]guid.Guid, 0), single_guid)
			min_distance = guid_distance
		} else if min_distance == guid_distance {
			retSlice = append(retSlice, single_guid)
		}
	}

	return retSlice, nil
}

func (cr *VivaldiCore[SUPPORT]) GetIsFailed(guid guid.Guid) bool {
	return cr.nodesCache[guid].IsFailed
}

func (cr *VivaldiCore[T]) GetCallback() func(rtt float64, metadata core.Metadata) {
	return cr.callback
}

func NewVivaldiCore[SUPPORT float64 | complex128](myGuid guid.Guid, myCoords []SUPPORT, space *nvs.NormedVectorSpace[SUPPORT]) (*VivaldiCore[SUPPORT], error) {

	if space.Dimension() == 0 {
		return nil, errors.New("space malformed, please use the New* function to properly initialize one")
	}
	space_coords, err := nvs.NewPoint(space, myCoords)
	if err != nil {
		return nil, errors.New("initial coordinate not compatible with the requested space")
	}

	cr := &VivaldiCore[SUPPORT]{
		nodesCache:    make(map[guid.Guid]NodeData[SUPPORT]),
		myCoordinates: space_coords,
		myGUID:        myGuid,
		space:         space,
	}

	cr.callback = func(rtt float64, metadata core.Metadata) {
		//The Vivaldi algorithm
	}

	return cr, nil
}
