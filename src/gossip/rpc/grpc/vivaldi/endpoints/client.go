package endpoints

import (
	"context"
	"errors"
	"fmt"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/core/impl/vivaldi"
	connectionmanager "github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/connection_manager"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
)

const core_code string = "Vivaldi"

type VivaldiRPCGossipClient struct {
	client pb_go.GossipStatusClient
	conn   *connectionmanager.GrpcCommunicationChannel
}

func NewVivaldiRPCGossipClient(peer guid.Guid, address string) (*VivaldiRPCGossipClient, error) {
	retVal := &VivaldiRPCGossipClient{}

	conn, err := connectionmanager.NewGrpcCommunicationChannel(peer, address)
	if err != nil {
		return nil, fmt.Errorf("error in obtaining connection for client, details: %s", err)
	}

	retVal.client = pb_go.NewGossipStatusClient(conn.Conn)

	return retVal, nil
}

func (vgc *VivaldiRPCGossipClient) Release() error {
	vgc.client = nil
	err := connectionmanager.InvalidateGrpcCommunicationChannel(vgc.conn)

	if err != nil {
		return fmt.Errorf("error deallocating vivaldi grpc gossip client usage, details: %s", err)
	}

	return nil
}

func asPointsFloat(updates vivaldi.VivaldiMetadata[float64]) []*pb_go.NodeState {
	retVal := make([]*pb_go.NodeState, 0)

	for k, v := range updates.Data {
		coordinates := v.Coords
		coordReal := &pb_go.CoordStream{Coords: coordinates}
		point := &pb_go.Point{Dimension: int64(len(coordinates)), CoordReal: coordReal}
		retVal = append(retVal, &pb_go.NodeState{Guid: k.String(), Coords: point, Failed: v.IsFailed})
	}

	return retVal
}

func asPointsCmplx(updates vivaldi.VivaldiMetadata[complex128]) []*pb_go.NodeState {
	retVal := make([]*pb_go.NodeState, 0)

	for k, v := range updates.Data {
		coordinates := v.Coords
		re_coords := make([]float64, 0)
		im_coords := make([]float64, 0)

		for _, coord := range coordinates {
			re_coords = append(re_coords, real(coord))
			im_coords = append(im_coords, imag(coord))
		}

		coordReal := &pb_go.CoordStream{Coords: re_coords}
		coordIm := &pb_go.CoordStream{Coords: im_coords}
		point := &pb_go.Point{Dimension: int64(len(coordinates)), CoordReal: coordReal, CoordIm: coordIm}
		retVal = append(retVal, &pb_go.NodeState{Guid: k.String(), Coords: point, Failed: v.IsFailed})
	}

	return retVal
}

func asNodeDataReal(array []*pb_go.NodeState) (map[guid.Guid]vivaldi.VivaldiMetaCoor[float64], error) {

	retVal := make(map[guid.Guid]vivaldi.VivaldiMetaCoor[float64])

	for i := 0; i < len(array); i++ {
		guid, err := guid.Deserialize([]byte(array[i].Guid))
		if err != nil {
			return nil, errors.New("error: wrong guid format")
		}

		nodeData := vivaldi.VivaldiMetaCoor[float64]{}
		nodeData.IsFailed = array[i].Failed
		nodeData.Coords = array[i].Coords.CoordReal.Coords

		retVal[guid] = nodeData
	}

	return retVal, nil
}

func asNodeDataCmplx(array []*pb_go.NodeState) (map[guid.Guid]vivaldi.VivaldiMetaCoor[complex128], error) {

	retVal := make(map[guid.Guid]vivaldi.VivaldiMetaCoor[complex128])

	for i := 0; i < len(array); i++ {
		guid, err := guid.Deserialize([]byte(array[i].Guid))
		if err != nil {
			return nil, errors.New("error deserializing a node's guid")
		}

		nodeData := vivaldi.VivaldiMetaCoor[complex128]{}
		nodeData.IsFailed = array[i].Failed

		cmplxCoords := make([]complex128, 0)

		for j := int64(0); j < array[i].Coords.Dimension; j++ {

			re := array[i].Coords.CoordReal.Coords[j]
			im := array[i].Coords.CoordIm.Coords[j]

			cmplxCoords = append(cmplxCoords, complex(re, im))
		}

		nodeData.Coords = cmplxCoords

		retVal[guid] = nodeData
	}

	return retVal, nil
}

func preparePush(nodeCore core.GNCFDCore) ([]*pb_go.NodeState, error) {

	if nodeCore.GetKind() != core_code {
		return nil, errors.New("error: the requested core is incompatible with this gossip client")
	}

	updates, err := nodeCore.GetStateUpdates()
	if err != nil {
		return nil, fmt.Errorf("unable to get state updates, details: %s", err)
	}

	var pointsToSend []*pb_go.NodeState
	switch updatedPoints := updates.(type) {
	case vivaldi.VivaldiMetadata[float64]:
		pointsToSend = asPointsFloat(updatedPoints)
	case vivaldi.VivaldiMetadata[complex128]:
		pointsToSend = asPointsCmplx(updatedPoints)
	default:
		return nil, errors.New("wrong metadata format")
	}

	return pointsToSend, nil
}

func executePull(nodeCore core.GNCFDCore, nodeUpdates *pb_go.NodeUpdates) error {

	guid, err := guid.Deserialize([]byte(nodeUpdates.CoreSession))
	if err != nil {
		return errors.New("error in deserializing session guid")
	}

	if nodeUpdates.Support == pb_go.Support_REAL {
		meta_data, err := asNodeDataReal(nodeUpdates.UpdatePayload)
		if err != nil {
			return fmt.Errorf("error in data translation, details: %s", err)
		}
		meta := vivaldi.VivaldiMetadata[float64]{Session: guid, Data: meta_data}
		err = nodeCore.UpdateState(meta)
		if err != nil {
			return fmt.Errorf("error in state update, details: %s", err)
		}
	} else if nodeUpdates.Support == pb_go.Support_CMPLX {
		meta_data, err := asNodeDataCmplx(nodeUpdates.UpdatePayload)
		if err != nil {
			return fmt.Errorf("error in data translation, details: %s", err)
		}
		meta := vivaldi.VivaldiMetadata[complex128]{Session: guid, Data: meta_data}
		err = nodeCore.UpdateState(meta)
		if err != nil {
			return fmt.Errorf("error in state update, details: %s", err)
		}
	} else {
		return fmt.Errorf("error: unknown support")
	}

	return nil
}

func (gc *VivaldiRPCGossipClient) Push(nodeCore core.GNCFDCore) error {

	pointsToSend, err := preparePush(nodeCore)
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}

	session := nodeCore.GetCoreSession().String()
	_, err = gc.client.PushGossip(context.Background(), &pb_go.NodeUpdates{CoreSession: session, UpdatePayload: pointsToSend})
	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	return nil
}

func (gc *VivaldiRPCGossipClient) Pull(nodeCore core.GNCFDCore) error {

	if nodeCore.GetKind() != core_code {
		return errors.New("error: the requested core is incompatible with this gossip client")
	}

	nodeUpdates, err := gc.client.PullGossip(context.Background(), &pb_go.CoreSession{CoreSession: nodeCore.GetCoreSession().String()})
	if err != nil {
		return fmt.Errorf("error in pull invocation, details: %s", err)
	}

	return executePull(nodeCore, nodeUpdates)
}

func (vgc *VivaldiRPCGossipClient) Exchange(nodeCore core.GNCFDCore) error {

	pointsToSend, err := preparePush(nodeCore)
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}

	session := nodeCore.GetCoreSession().String()
	nodeUpdates, err := vgc.client.ExchangeGossip(context.Background(), &pb_go.NodeUpdates{CoreSession: session, UpdatePayload: pointsToSend})
	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	return executePull(nodeCore, nodeUpdates)
}
