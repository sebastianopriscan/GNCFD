//go:build !release
// +build !release

package vivaldi

import (
	"errors"
	"fmt"

	//LOG_PUSH
	"log"
	//LOG_POP
	"math"
	"sync"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/nvs"
	channelobserver "github.com/sebastianopriscan/GNCFD/utils/channel_observer"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
)

type nodeData[SUPPORT float64 | complex128] struct {
	IsFailed bool
	Coords   *nvs.Point[SUPPORT]
	Updated  bool
}

type VivaldiCore[SUPPORT float64 | complex128] struct {
	channelobserver.ChannelObserverSubjectImpl

	sess_mu sync.RWMutex
	core_mu sync.RWMutex

	nodesCache    map[guid.Guid]*nodeData[SUPPORT]
	myGUID        guid.Guid
	myCoordinates *nvs.Point[SUPPORT]
	space         *nvs.NormedVectorSpace[SUPPORT]

	session guid.Guid

	ce float64
	cc float64
	ei float64
}

func (cr *VivaldiCore[SUPPORT]) GetClosestOf(guids []guid.Guid) ([]guid.Guid, error) {
	min_distance := math.MaxFloat64
	var retSlice []guid.Guid

	cr.core_mu.RLock()
	defer cr.core_mu.RUnlock()

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
	cr.core_mu.RLock()
	defer cr.core_mu.RUnlock()
	return cr.nodesCache[guid].IsFailed
}

func (cr *VivaldiCore[SUPPORT]) GetCoreSession() guid.Guid {
	cr.sess_mu.RLock()
	defer cr.sess_mu.RUnlock()
	return cr.session
}

func (cr *VivaldiCore[SUPPORT]) SetCoreSession(guid guid.Guid) {
	cr.sess_mu.Lock()
	defer cr.sess_mu.Unlock()
	cr.session = guid
}

func (cr *VivaldiCore[SUPPORT]) GetKind() string {
	return "Vivaldi"
}

func (cr *VivaldiCore[SUPPORT]) GetStateUpdates() (core.Metadata, error) {
	cr.core_mu.RLock()
	defer cr.core_mu.RUnlock()

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

func (cr *VivaldiCore[SUPPORT]) updatePoint(point *nvs.Point[SUPPORT], newCoords []SUPPORT) error {
	newCoordsPoint, err := nvs.NewPoint(cr.space, newCoords)
	if err != nil {
		return fmt.Errorf("error generating point updates, details: %s", err)
	}
	newCoordsPointHalf, err := cr.space.ExternalMul(newCoordsPoint, 0.5)
	if err != nil {
		return fmt.Errorf("error generating point updates, details: %s", err)
	}
	origPointHalf, err := cr.space.ExternalMul(point, 0.5)
	if err != nil {
		return fmt.Errorf("error generating point updates, details: %s", err)
	}

	newCoordsSliceHalf := newCoordsPointHalf.GetCoordinates()
	origSliceHalf := origPointHalf.GetCoordinates()
	sum := make([]SUPPORT, cr.space.Dimension())
	for i := 0; i < cr.space.Dimension(); i++ {
		sum[i] = newCoordsSliceHalf[i] + origSliceHalf[i]
	}

	res := point.SetCoordinates(sum)
	if !res {
		return errors.New("error setting coordinates for point, dimension/support not compatible")
	}

	return nil
}

func (cr *VivaldiCore[SUPPORT]) vivaldi_update(rtt float64, ej float64, communicator guid.Guid) {

	//DEBUG_PUSH
	mssg := "Vivaldi Core: running vivaldi_update:\n"
	//DEBUG_POP

	var w float64
	if cr.ei+ej != 0 {
		w = cr.ei / (cr.ei + ej)
	} else {
		w = 10e-5 //Justified by the fact that if errors converge to 0, the algorithm loses its adaptability
	}

	//DEBUG_PUSH
	mssg += fmt.Sprintf("\tcr.ei = %v\n\tej = %v\n\tw = %v\n", cr.ei, ej, w)
	//DEBUG_POP

	commData, present := cr.nodesCache[communicator]
	if !present {
		//DEBUG_PUSH
		mssg += "\tCommunicator not present in table, returning"
		log.Print(mssg)
		//DEBUG_POP
		return
	}
	commCoords := commData.Coords

	//DEBUG_PUSH
	mssg += "\tMy Coordinates:\n"
	for _, coor := range cr.myCoordinates.GetCoordinates() {
		mssg += fmt.Sprintf("\t\t%v\n", coor)
	}

	mssg += fmt.Sprintf("\tCommunicator GUID: %v\n\tCommunicator coordinates:\n", communicator)
	for _, coord := range commCoords.GetCoordinates() {
		mssg += fmt.Sprintf("\t\t%v\n", coord)
	}
	//DEBUG_POP

	dist, err := cr.space.Distance(cr.myCoordinates, commCoords)
	if err != nil {
		//DEBUG_PUSH
		mssg += "\tError generating coordinates, details: " + err.Error() + "\n"
		log.Print(mssg)
		//DEBUG_POP
		return
	}

	//DEBUG_PUSH
	mssg += fmt.Sprintf("\tDistance: %v\n", dist)
	//DEBUG_POP

	e := (rtt - dist)
	es := math.Abs(e) / rtt

	//DEBUG_PUSH
	mssg += fmt.Sprintf("\te = %v\n\tes = %v\n\trtt = %v", e, es, rtt)
	//DEBUG_POP

	cr.ei = es * cr.ce * w * cr.ei * (1 - cr.ce*w)
	delta := cr.cc * w

	//DEBUG_PUSH
	mssg += fmt.Sprintf("\t*cr.ei = %v\n\tdelta = %v\n", cr.ei, delta)
	//DEBUG_POP

	unit, err := cr.space.UnitVector(cr.myCoordinates, commCoords)
	if err != nil {
		//DEBUG_PUSH
		mssg += fmt.Sprintf("\tError generating Unit vector, details: " + err.Error() + "\n")
		log.Print(mssg)
		//DEBUG_POP
		return
	}

	//DEBUG_PUSH
	mssg += "\tUnit vector coordinates:\n"
	for _, coor := range unit.GetCoordinates() {
		mssg += fmt.Sprintf("\t\t%v\n", coor)
	}
	//DEBUG_POP

	mulPt, err := cr.space.ExternalMul(unit, e*delta)
	if err != nil {
		//DEBUG_PUSH
		mssg += "\tError doing external mul, details: " + err.Error() + "\n"
		log.Print(mssg)
		//DEBUG_POP
		return
	}

	//DEBUG_PUSH
	mssg += "\tExMul vector coordinates:\n"
	for _, coor := range mulPt.GetCoordinates() {
		mssg += fmt.Sprintf("\t\t%v\n", coor)
	}
	//DEBUG_POP

	newCoordinates := make([]SUPPORT, cr.space.Dimension())
	myCoords := cr.myCoordinates.GetCoordinates()
	mulPt_coors := mulPt.GetCoordinates()

	//DEBUG_PUSH
	mssg += "\tSelf new coordinates:\n"
	//DEBUG_POP
	for i := 0; i < cr.space.Dimension(); i++ {
		newCoordinates[i] = myCoords[i] + mulPt_coors[i]
		//DEBUG_PUSH
		mssg += fmt.Sprintf("\t\t%v\n", newCoordinates[i])
		//DEBUG_POP
	}

	cr.myCoordinates.SetCoordinates(newCoordinates)

	//DEBUG_PUSH
	log.Print(mssg)
	//DEBUG_POP
}

func (cr *VivaldiCore[SUPPORT]) GetMyState() (core.Metadata, error) {
	cr.core_mu.RLock()
	defer cr.core_mu.RUnlock()
	return &VivaldiPeerState[SUPPORT]{Me: cr.myGUID, Coords: cr.myCoordinates.GetCoordinates(), Ej: cr.ei}, nil
}

//DUMPOINT_PUSH

var guidLogs map[guid.Guid]int = make(map[guid.Guid]int)

//DUMPOINT_POP

func (cr *VivaldiCore[SUPPORT]) UpdateState(metadata core.Metadata) error {
	nodes, ok := metadata.(*VivaldiMetadata[SUPPORT])
	if !ok {
		return errors.New("error: bad metadata passed")
	}

	cr.sess_mu.RLock()
	if nodes.Session != cr.session {
		cr.sess_mu.RUnlock()
		return errors.New("error : incompatible core session")
	}
	cr.sess_mu.RUnlock()

	cr.core_mu.Lock()
	defer cr.core_mu.Unlock()

	var err error = nil
	for extGuid, data := range nodes.Data {
		node, present := cr.nodesCache[extGuid]
		//DUMPOINT_PUSH
		_, guidpres := guidLogs[extGuid]
		if !guidpres {
			guidLogs[extGuid] = 0
		}
		if guidLogs[extGuid] != 4 {
			coorMsg := ""
			coorMsg += fmt.Sprintf("Dumping info for GUID %v:\n\tCoordinates:\n", extGuid)
			for _, coor := range data.Coords {
				coorMsg += fmt.Sprintf("\t\t%v\n", coor)
			}
			log.Print(coorMsg)
			guidLogs[extGuid]++
		}

		//DUMPOINT_POP
		if present {
			node.IsFailed = data.IsFailed
			cr.updatePoint(node.Coords, data.Coords)
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

			cr.nodesCache[extGuid] = node
		}
	}

	cr.vivaldi_update(nodes.Rtt, nodes.Ej, nodes.Communicator)

	//Classical Observer notify, the observers will keep a reference to the core to get the updates
	//cr.PushToChannels(true)

	return err
}

func (cr *VivaldiCore[SUPPORT]) SignalFailed(peers []guid.Guid) {
	cr.core_mu.Lock()
	defer cr.core_mu.Unlock()

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
		ei:            10.,

		ChannelObserverSubjectImpl: channelobserver.NewChannelObserverSubjectImpl(),
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

type VivaldiPeerState[SUPPORT float64 | complex128] struct {
	Me     guid.Guid
	Coords []SUPPORT
	Ej     float64
}

//DUMP_PUSH

func (cr *VivaldiCore[SUPPORT]) DumpCore() (*VivaldiMetadata[SUPPORT], error) {

	cr.core_mu.RLock()
	defer cr.core_mu.RUnlock()

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

		data[k] = VivaldiMetaCoor[SUPPORT]{
			IsFailed: v.IsFailed,
			Coords:   v.Coords.GetCoordinates(),
		}
	}

	retVal.Data = data

	return retVal, nil
}

//DUMP_POP
