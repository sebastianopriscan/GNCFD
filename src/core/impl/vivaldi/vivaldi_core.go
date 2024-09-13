package vivaldi

import (
	"errors"
	"fmt"
	"math"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/core/nvs"
	"github.com/sebastianopriscan/GNCFD/gossip"
)

type nodeData[SUPPORT float64 | complex128] struct {
	IsFailed bool
	Coords   *nvs.Point[SUPPORT]
	Updated  bool
}

type VivaldiCore[SUPPORT float64 | complex128] struct {
	gossip.ChannelObserverSubjectImpl

	nodesCache    map[guid.Guid]*nodeData[SUPPORT]
	myGUID        guid.Guid
	myCoordinates *nvs.Point[SUPPORT]
	space         *nvs.NormedVectorSpace[SUPPORT]

	session guid.Guid

	ce float64
	cc float64
	ei float64
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

func (cr *VivaldiCore[SUPPORT]) GetCoreSession() guid.Guid {
	return cr.session
}

func (cr *VivaldiCore[SUPPORT]) SetCoreSession(guid guid.Guid) {
	cr.session = guid
}

func (cr *VivaldiCore[SUPPORT]) GetKind() string {
	return "Vivaldi"
}

func (cr *VivaldiCore[SUPPORT]) GetStateUpdates() (core.Metadata, error) {

	retVal := &VivaldiMetadata[SUPPORT]{
		Session:      cr.session,
		Ej:           cr.ei,
		Communicator: cr.myGUID,
	}

	data := make(map[guid.Guid]VivaldiMetaCoor[SUPPORT])

	data[cr.myGUID] = VivaldiMetaCoor[SUPPORT]{
		IsFailed: false,
		Coords:   cr.myCoordinates.GetCoordinates(),
	}

	for k, v := range cr.nodesCache {
		if v.Updated {
			data[k] = VivaldiMetaCoor[SUPPORT]{
				IsFailed: v.IsFailed,
				Coords:   v.Coords.GetCoordinates(),
			}

			v.Updated = false
		}
	}

	retVal.Data = data

	return retVal, nil
}

func updatePoint[SUPPORT float64 | complex128](*nvs.Point[SUPPORT], []SUPPORT) {
	//TODO : Decide if a general coordinate update should be needed and eventually customizable
}

func (cr *VivaldiCore[SUPPORT]) vivaldi_update(rtt float64, ej float64, communicator guid.Guid) {

	w := cr.ei / (cr.ei + ej)

	commData, present := cr.nodesCache[communicator]
	if !present {
		return
	}
	commCoords := commData.Coords

	dist, err := cr.space.Distance(cr.myCoordinates, commCoords)
	if err != nil {
		return
	}

	e := (rtt - dist)
	es := math.Abs(e) / rtt

	cr.ei = es * cr.ce * w * cr.ei * (1 - cr.ce*w)
	delta := cr.cc * w

	unit, err := cr.space.UnitVector(cr.myCoordinates, commCoords)
	if err != nil {
		return
	}

	mulPt, err := cr.space.ExternalMul(unit, e*delta)
	if err != nil {
		return
	}

	newCoordinates := make([]SUPPORT, cr.space.Dimension())
	myCoords := cr.myCoordinates.GetCoordinates()
	mulPt_coors := mulPt.GetCoordinates()

	for i := 0; i < cr.space.Dimension(); i++ {

		newCoordinates[i] = myCoords[i] + mulPt_coors[i]
	}

	cr.myCoordinates.SetCoordinates(newCoordinates)
}

func (cr *VivaldiCore[SUPPORT]) UpdateState(metadata core.Metadata) error {
	nodes, ok := metadata.(VivaldiMetadata[SUPPORT])
	if !ok {
		return errors.New("error: bad metadata passed")
	}

	if nodes.Session != cr.session {
		return errors.New("error : incompatible core session")
	}

	var err error = nil
	for guid, data := range nodes.Data {
		node, present := cr.nodesCache[guid]
		if present {
			node.IsFailed = data.IsFailed
			updatePoint(node.Coords, data.Coords)
			node.Updated = true
		} else {

			var point *nvs.Point[SUPPORT]
			point, err = nvs.NewPoint(cr.space, data.Coords)
			if err != nil {
				err = fmt.Errorf("error : at least an error has been encountered, details : %s", err)
				continue
			}

			node = &nodeData[SUPPORT]{
				IsFailed: data.IsFailed,
				Updated:  true,
				Coords:   point,
			}

			cr.nodesCache[guid] = node
		}
	}

	cr.vivaldi_update(nodes.Rtt, nodes.Ej, nodes.Communicator)

	//Classical Observer notify, the observers will keep a reference to the core to get the updates
	cr.PushToChannels(true)

	return err
}

func (cr *VivaldiCore[SUPPORT]) SignalFailed(peers []guid.Guid) {
	for _, peer := range peers {
		data, present := cr.nodesCache[peer]
		if !present {
			continue
		}
		data.IsFailed = true
		data.Updated = true
	}
}

func NewVivaldiCore[SUPPORT float64 | complex128](myGuid guid.Guid, myCoords []SUPPORT, space *nvs.NormedVectorSpace[SUPPORT],
	ce float64, cc float64) (*VivaldiCore[SUPPORT], error) {

	if space.Dimension() == 0 {
		return nil, errors.New("space malformed, please use the New* function to properly initialize one")
	}
	space_coords, err := nvs.NewPoint(space, myCoords)
	if err != nil {
		return nil, errors.New("initial coordinate not compatible with the requested space")
	}

	cr := &VivaldiCore[SUPPORT]{
		nodesCache:    make(map[guid.Guid]*nodeData[SUPPORT]),
		myCoordinates: space_coords,
		myGUID:        myGuid,
		space:         space,
		ce:            ce,
		cc:            cc,
		ei:            0.,
	}

	return cr, nil
}

type VivaldiMetaCoor[SUPPORT float64 | complex128] struct {
	IsFailed bool
	Coords   []SUPPORT
}

type VivaldiMetadata[SUPPORT float64 | complex128] struct {
	Session guid.Guid
	Data    map[guid.Guid]VivaldiMetaCoor[SUPPORT]

	Rtt          float64
	Ej           float64
	Communicator guid.Guid
}
