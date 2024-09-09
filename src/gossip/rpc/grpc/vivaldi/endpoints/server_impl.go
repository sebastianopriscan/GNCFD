package endpoints

import (
	"context"
	"errors"
	"fmt"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/core/impl/vivaldi"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
)

type VivaldiGRPCGossipServer struct {
	pb_go.UnimplementedGossipStatusServer
	coreMap map[guid.Guid]core.GNCFDCore
}

func do_push_gossip(nodes *pb_go.NodeUpdates, core core.GNCFDCore, guid guid.Guid) (*pb_go.PushReturn, error) {

	switch nodes.Support {
	case pb_go.Support_REAL:
		updates, err := asNodeDataReal(nodes.UpdatePayload)
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in data conversion, push failed, details: %s", err)
		}
		err = core.UpdateState(&vivaldi.VivaldiMetadata[float64]{Session: guid, Data: updates})
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in core upadate, push failed, details: %s", err)
		}
	case pb_go.Support_CMPLX:
		updates, err := asNodeDataCmplx(nodes.UpdatePayload)
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in data conversion, push failed, details: %s", err)
		}
		err = core.UpdateState(&vivaldi.VivaldiMetadata[complex128]{Session: guid, Data: updates})
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in core upadate, push failed, details: %s", err)
		}
	default:
		return &pb_go.PushReturn{}, errors.New("error: wrong nvs support, push failed")
	}

	return &pb_go.PushReturn{}, nil
}

func do_pull_gossip(core core.GNCFDCore) (*pb_go.NodeUpdates, error) {

	updates, err := core.GetStateUpdates()
	if err != nil {
		return nil, errors.New("error in getting core updates, pull failed")
	}

	var pointsToSend pb_go.NodeUpdates
	pointsToSend.CoreSession = core.GetCoreSession().String()

	switch updatedPoints := updates.(type) {
	case vivaldi.VivaldiMetadata[float64]:
		pointsToSend.Support = pb_go.Support_REAL
		pointsToSend.UpdatePayload = asPointsFloat(updatedPoints)
	case vivaldi.VivaldiMetadata[complex128]:
		pointsToSend.Support = pb_go.Support_CMPLX
		pointsToSend.UpdatePayload = asPointsCmplx(updatedPoints)
	default:
		return nil, errors.New("wrong metadata format")
	}

	return &pointsToSend, nil
}

func (vgs *VivaldiGRPCGossipServer) PushGossip(ctx context.Context, nodes *pb_go.NodeUpdates) (*pb_go.PushReturn, error) {

	guid, err := guid.Deserialize([]byte(nodes.CoreSession))
	if err != nil {
		return &pb_go.PushReturn{}, errors.New("error converting guid, push failed")
	}

	core, ok := vgs.coreMap[guid]
	if !ok {
		return &pb_go.PushReturn{}, errors.New("error: no core with such session, push failed")
	}

	if core.GetKind() != core_code {
		return &pb_go.PushReturn{}, errors.New("error: requested core incompatible with sender one, push failed")
	}

	return do_push_gossip(nodes, core, guid)
}

func (vgs *VivaldiGRPCGossipServer) PullGossip(ctx context.Context, session *pb_go.CoreSession) (*pb_go.NodeUpdates, error) {

	guid, err := guid.Deserialize([]byte(session.CoreSession))
	if err != nil {
		return nil, errors.New("error converting guid, pull failed")
	}

	core, ok := vgs.coreMap[guid]
	if !ok {
		return nil, errors.New("error: no core with such session, pull failed")
	}

	if core.GetKind() != core_code {
		return nil, errors.New("error: requested core incompatible with sender one, pull failed")
	}

	return do_pull_gossip(core)
}

func (vgs *VivaldiGRPCGossipServer) ExchangeGossip(ctx context.Context, nodes *pb_go.NodeUpdates) (*pb_go.NodeUpdates, error) {

	guid, err := guid.Deserialize([]byte(nodes.CoreSession))
	if err != nil {
		return nil, errors.New("error converting guid, pull failed")
	}

	core, ok := vgs.coreMap[guid]
	if !ok {
		return nil, errors.New("error: no core with such session, pull failed")
	}

	if core.GetKind() != core_code {
		return nil, errors.New("error: requested core incompatible with sender one, pull failed")
	}

	_, err = do_push_gossip(nodes, core, guid)
	if err != nil {
		return nil, fmt.Errorf("error: unable to push gossip, exchange failed, details: %s", err)
	}

	return do_pull_gossip(core)
}
