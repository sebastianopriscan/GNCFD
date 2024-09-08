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

func asPointsFloat(session string, updates map[guid.Guid]vivaldi.NodeData[float64]) []*pb_go.NodeState {
	retVal := make([]*pb_go.NodeState, 0)

	for k, v := range updates {
		coordinates := v.Coords.GetCoordinates()
		coordReal := &pb_go.CoordStream{Coords: coordinates}
		point := &pb_go.Point{Dimension: int64(len(coordinates)), CoordReal: coordReal, Support: pb_go.Support_REAL}
		retVal = append(retVal, &pb_go.NodeState{Guid: k.String(), Coords: point, Failed: v.IsFailed})
	}

	return retVal
}

func asPointsCmplx(session string, updates map[guid.Guid]vivaldi.NodeData[complex128]) []*pb_go.NodeState {
	retVal := make([]*pb_go.NodeState, 0)

	for k, v := range updates {
		coordinates := v.Coords.GetCoordinates()
		re_coords := make([]float64, 0)
		im_coords := make([]float64, 0)

		for _, coord := range coordinates {
			re_coords = append(re_coords, real(coord))
			im_coords = append(im_coords, imag(coord))
		}

		coordReal := &pb_go.CoordStream{Coords: re_coords}
		coordIm := &pb_go.CoordStream{Coords: im_coords}
		point := &pb_go.Point{Dimension: int64(len(coordinates)), CoordReal: coordReal, CoordIm: coordIm, Support: pb_go.Support_CMPLX}
		retVal = append(retVal, &pb_go.NodeState{Guid: k.String(), Coords: point, Failed: v.IsFailed})
	}

	return retVal
}

func (gc *VivaldiRPCGossipClient) Push(nodeCore core.GNCFDCore) error {

	if nodeCore.GetKind() != core_code {
		return errors.New("error: the requested core is incompatible with this gossip client")
	}

	updates, err := nodeCore.GetStateUpdates()
	if err != nil {
		return fmt.Errorf("unable to get state updates, details: %s", err)
	}

	var pointsToSend []*pb_go.NodeState
	switch updatedPoints := updates.(type) {
	case map[guid.Guid]vivaldi.NodeData[float64]:
		pointsToSend = asPointsFloat(nodeCore.GetCoreSession().String(), updatedPoints)
	case map[guid.Guid]vivaldi.NodeData[complex128]:
		pointsToSend = asPointsCmplx(nodeCore.GetCoreSession().String(), updatedPoints)
	default:
		return errors.New("wrong metadata format")
	}

	session := nodeCore.GetCoreSession().String()
	_, err = gc.client.PushGossip(context.Background(), &pb_go.NodeUpdates{CoreSession: &session, UpdatePayload: pointsToSend})
	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	return nil
}

func (gc *VivaldiRPCGossipClient) Pull(nodeCore core.GNCFDCore) error {

	nodeUpdates, err := gc.client.PullGossip(context.Background(), &pb_go.CoreSession{CoreSession: nodeCore.GetCoreSession().String()})
	if err != nil {
		return fmt.Errorf("error in pull invocation, details: %s", err)
	}

	for _, update := range nodeUpdates.UpdatePayload {

	}
}
