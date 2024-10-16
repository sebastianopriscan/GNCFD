package endpoints

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/sebastianopriscan/GNCFD/communication/rpc/grpc/vivaldi/pb_go"
	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/impl/vivaldi"
	"github.com/sebastianopriscan/GNCFD/gossip"
	channelobserver "github.com/sebastianopriscan/GNCFD/utils/channel_observer"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
	lockedmap "github.com/sebastianopriscan/GNCFD/utils/locked_map"
	"github.com/sebastianopriscan/GNCFD/utils/ntptime"
)

type VivaldiGRPCGossipServer struct {
	channelobserver.ChannelObserverSubjectImpl
	pb_go.UnimplementedGossipStatusServer
	coreMap *lockedmap.LockedMap[guid.Guid, core.GNCFDCoreInteractionGate]
}

func do_push_gossip(nodes *pb_go.NodeUpdates, core core.GNCFDCoreInteractionGate, sessGuid guid.Guid, now int64) (*pb_go.PushReturn, error) {

	sender, err := guid.Deserialize([]byte(nodes.Sender))
	if err != nil {
		return &pb_go.PushReturn{}, errors.New("error converting guid, push failed")
	}

	switch nodes.Support {
	case pb_go.Support_REAL:
		updates, err := asNodeDataReal(nodes.UpdatePayload)
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in data conversion, push failed, details: %s", err)
		}

		err = core.UpdateState(&vivaldi.VivaldiMetadata[float64]{
			Session:      sessGuid,
			Data:         updates,
			Communicator: sender,
			Rtt:          math.Abs(float64(now-nodes.Timestamp)) / 2.0,
			Ej:           nodes.Ej,
		})
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in core upadate, push failed, details: %s", err)
		}
	case pb_go.Support_CMPLX:
		updates, err := asNodeDataCmplx(nodes.UpdatePayload)
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in data conversion, push failed, details: %s", err)
		}
		err = core.UpdateState(&vivaldi.VivaldiMetadata[complex128]{
			Session:      sessGuid,
			Data:         updates,
			Communicator: sender,
			Rtt:          math.Abs(float64(now-nodes.Timestamp)) / 2.0,
			Ej:           nodes.Ej,
		})
		if err != nil {
			return &pb_go.PushReturn{}, fmt.Errorf("error in core upadate, push failed, details: %s", err)
		}
	default:
		return &pb_go.PushReturn{}, errors.New("error: wrong nvs support, push failed")
	}

	return &pb_go.PushReturn{}, nil
}

func do_pull_gossip(core core.GNCFDCoreInteractionGate) (*pb_go.NodeUpdates, error) {

	updates, err := core.GetStateUpdates()
	if err != nil {
		return nil, errors.New("error in getting core updates, pull failed")
	}

	var pointsToSend pb_go.NodeUpdates
	pointsToSend.CoreSession = core.GetCoreSession().String()

	messID, err := guid.GenerateGUID()
	if err != nil {
		return nil, fmt.Errorf("error in message ID geenration, datails: %s", err)
	}

	pointsToSend.MessageID = messID.String()

	switch updatedPoints := updates.(type) {
	case *vivaldi.VivaldiMetadata[float64]:
		pointsToSend.Support = pb_go.Support_REAL
		pointsToSend.Sender = updatedPoints.Communicator.String()
		pointsToSend.Ej = updatedPoints.Ej
		pointsToSend.UpdatePayload = asPointsFloat(updatedPoints)
	case *vivaldi.VivaldiMetadata[complex128]:
		pointsToSend.Support = pb_go.Support_CMPLX
		pointsToSend.Sender = updatedPoints.Communicator.String()
		pointsToSend.Ej = updatedPoints.Ej
		pointsToSend.UpdatePayload = asPointsCmplx(updatedPoints)
	default:
		return nil, errors.New("wrong metadata format")
	}

	time, err := ntptime.GetNTPTime()
	if err != nil {
		return nil, fmt.Errorf("error in parameters preparation, details: %s", err)
	}
	pointsToSend.Timestamp = time.UnixNano()

	return &pointsToSend, nil
}

func (vgs *VivaldiGRPCGossipServer) PushGossip(ctx context.Context, nodes *pb_go.NodeUpdates) (*pb_go.PushReturn, error) {

	nowTime, err := ntptime.GetNTPTime()
	if err != nil {
		return &pb_go.PushReturn{}, fmt.Errorf("error in timestamp creation, details: %s", err)
	}
	now := nowTime.UnixNano()

	sessGuid, err := guid.Deserialize([]byte(nodes.CoreSession))
	if err != nil {
		return &pb_go.PushReturn{}, errors.New("error converting guid, push failed")
	}

	vgs.coreMap.Mu.RLock()
	defer vgs.coreMap.Mu.RUnlock()

	core, ok := vgs.coreMap.Map[sessGuid]
	if !ok {
		return &pb_go.PushReturn{}, errors.New("error: no core with such session, push failed")
	}

	if core.GetKind() != core_code {
		return &pb_go.PushReturn{}, errors.New("error: requested core incompatible with sender one, push failed")
	}

	msgID, err := guid.Deserialize([]byte(nodes.MessageID))
	if err != nil {
		return &pb_go.PushReturn{}, errors.New("error deserializing message_id")
	}

	sender, err := guid.Deserialize([]byte(nodes.Sender))
	if err != nil {
		return &pb_go.PushReturn{}, errors.New("error deserializing sender")
	}

	//Pushing updates to channels
	vgs.PushToChannels(&gossip.MessageToForward{MessageID: msgID, Sender: sender, Payload: nodes})

	return do_push_gossip(nodes, core, sessGuid, now)
}

func (vgs *VivaldiGRPCGossipServer) PullGossip(ctx context.Context, session *pb_go.CoreSession) (*pb_go.NodeUpdates, error) {

	guid, err := guid.Deserialize([]byte(session.CoreSession))
	if err != nil {
		return nil, errors.New("error converting guid, pull failed")
	}

	vgs.coreMap.Mu.RLock()
	defer vgs.coreMap.Mu.RUnlock()

	core, ok := vgs.coreMap.Map[guid]
	if !ok {
		return nil, errors.New("error: no core with such session, pull failed")
	}

	if core.GetKind() != core_code {
		return nil, errors.New("error: requested core incompatible with sender one, pull failed")
	}

	return do_pull_gossip(core)
}

func (vgs *VivaldiGRPCGossipServer) ExchangeGossip(ctx context.Context, nodes *pb_go.NodeUpdates) (*pb_go.NodeUpdates, error) {

	nowTime, err := ntptime.GetNTPTime()
	if err != nil {
		return nil, fmt.Errorf("error in timestamp creation, details: %s", err)
	}
	now := nowTime.UnixNano()

	guid, err := guid.Deserialize([]byte(nodes.CoreSession))
	if err != nil {
		return nil, errors.New("error converting guid, pull failed")
	}

	vgs.coreMap.Mu.RLock()
	defer vgs.coreMap.Mu.RUnlock()

	core, ok := vgs.coreMap.Map[guid]
	if !ok {
		return nil, errors.New("error: no core with such session, pull failed")
	}

	if core.GetKind() != core_code {
		return nil, errors.New("error: requested core incompatible with sender one, pull failed")
	}

	_, err = do_push_gossip(nodes, core, guid, now)
	if err != nil {
		return nil, fmt.Errorf("error: unable to push gossip, exchange failed, details: %s", err)
	}

	return do_pull_gossip(core)
}
