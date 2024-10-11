package endpoints

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/impl/vivaldi"
	connectionmanager "github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/connection_manager"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
	"github.com/sebastianopriscan/GNCFD/utils/ntptime"
)

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
	retVal.conn = conn

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

func preparePush(nodeCore core.GNCFDCoreInteractionGate, updates core.CoreData) (*pb_go.NodeUpdates, error) {

	if nodeCore.GetKind() != core_code {
		return nil, errors.New("error: the requested core is incompatible with this gossip client")
	}

	var pointsToSend pb_go.NodeUpdates
	pointsToSend.CoreSession = nodeCore.GetCoreSession().String()

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

	return &pointsToSend, nil
}

func executePull(nodeCore core.GNCFDCoreInteractionGate, nodeUpdates *pb_go.NodeUpdates, time int64) error {

	sessGuid, err := guid.Deserialize([]byte(nodeUpdates.CoreSession))
	if err != nil {
		return errors.New("error in deserializing session guid")
	}

	sender, err := guid.Deserialize([]byte(nodeUpdates.Sender))
	if err != nil {
		return errors.New("error in deserializing sender guid")
	}

	if nodeUpdates.Support == pb_go.Support_REAL {
		meta_data, err := asNodeDataReal(nodeUpdates.UpdatePayload)
		if err != nil {
			return fmt.Errorf("error in data translation, details: %s", err)
		}
		meta := vivaldi.VivaldiMetadata[float64]{
			Session:      sessGuid,
			Data:         meta_data,
			Rtt:          math.Abs(float64(time - nodeUpdates.Timestamp)),
			Communicator: sender,
			Ej:           nodeUpdates.Ej,
		}
		err = nodeCore.UpdateState(meta)
		if err != nil {
			return fmt.Errorf("error in state update, details: %s", err)
		}
	} else if nodeUpdates.Support == pb_go.Support_CMPLX {
		meta_data, err := asNodeDataCmplx(nodeUpdates.UpdatePayload)
		if err != nil {
			return fmt.Errorf("error in data translation, details: %s", err)
		}
		meta := vivaldi.VivaldiMetadata[complex128]{
			Session:      sessGuid,
			Data:         meta_data,
			Rtt:          math.Abs(float64(time - nodeUpdates.Timestamp)),
			Communicator: sender,
			Ej:           nodeUpdates.Ej,
		}
		err = nodeCore.UpdateState(meta)
		if err != nil {
			return fmt.Errorf("error in state update, details: %s", err)
		}
	} else {
		return fmt.Errorf("error: unknown support")
	}

	return nil
}

func (gc *VivaldiRPCGossipClient) Push(nodeCore core.GNCFDCoreInteractionGate, coreData core.CoreData, messageID guid.Guid) error {

	pointsToSend, err := preparePush(nodeCore, coreData)
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}

	pointsToSend.MessageID = messageID.String()

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	time, err := ntptime.GetNTPTime()
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}
	pointsToSend.Timestamp = time.UnixNano()

	_, err = gc.client.PushGossip(timeout, pointsToSend)
	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	return nil
}

func (gc *VivaldiRPCGossipClient) Pull(nodeCore core.GNCFDCoreInteractionGate) error {

	if nodeCore.GetKind() != core_code {
		return errors.New("error: the requested core is incompatible with this gossip client")
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodeUpdates, err := gc.client.PullGossip(timeout, &pb_go.CoreSession{CoreSession: nodeCore.GetCoreSession().String()})
	if err != nil {
		return fmt.Errorf("error in pull invocation, details: %s", err)
	}

	nowTime, err := ntptime.GetNTPTime()
	if err != nil {
		return fmt.Errorf("error in timestamp creation, details: %s", err)
	}
	now := nowTime.UnixNano()

	return executePull(nodeCore, nodeUpdates, now)
}

func (vgc *VivaldiRPCGossipClient) Exchange(nodeCore core.GNCFDCoreInteractionGate, coreData core.CoreData, messageID guid.Guid) error {

	pointsToSend, err := preparePush(nodeCore, coreData)
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}

	pointsToSend.MessageID = messageID.String()

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	time, err := ntptime.GetNTPTime()
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}
	pointsToSend.Timestamp = time.UnixNano()

	nodeUpdates, err := vgc.client.ExchangeGossip(timeout, pointsToSend)
	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	time, err = ntptime.GetNTPTime()
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}

	return executePull(nodeCore, nodeUpdates, time.UnixNano())
}

func (vgc *VivaldiRPCGossipClient) Forward(nodeCore core.GNCFDCoreInteractionGate, data core.CoreData) error {

	if nodeCore.GetKind() != core_code {
		return errors.New("error: the requested core is incompatible with this gossip client")
	}

	nodes, ok := data.(*pb_go.NodeUpdates)
	if !ok {
		return errors.New("error: bad message passed")
	}

	coreStatus, _ := nodeCore.GetMyState()
	switch coreStatusReal := coreStatus.(type) {
	case *vivaldi.VivaldiPeerState[float64]:
		if nodes.Support == pb_go.Support_REAL {
			coordinates := coreStatusReal.Coords
			point := asPointFloat(coordinates)
			found := false
			for _, nodeState := range nodes.UpdatePayload {
				if nodeState.Guid == coreStatusReal.Me.String() {
					nodeState.Coords = point
					found = true
					break
				}
			}
			if !found {
				toAppend := &pb_go.NodeState{
					Guid:   coreStatusReal.Me.String(),
					Failed: false,
					Coords: point}
				nodes.UpdatePayload = append(nodes.UpdatePayload, toAppend)
			}
			nodes.Sender = coreStatusReal.Me.String()
			nodes.Ej = coreStatusReal.Ej
		} else {
			return errors.New("error: bad message passed (incompatible support)")
		}
	case *vivaldi.VivaldiPeerState[complex128]:

		if nodes.Support == pb_go.Support_CMPLX {
			coordinates := coreStatusReal.Coords
			point := asPointCmplx(coordinates)
			found := false
			for _, nodeState := range nodes.UpdatePayload {
				if nodeState.Guid == coreStatusReal.Me.String() {
					nodeState.Coords = point
					found = true
					break
				}
			}
			if !found {
				toAppend := &pb_go.NodeState{
					Guid:   coreStatusReal.Me.String(),
					Failed: false,
					Coords: point}
				nodes.UpdatePayload = append(nodes.UpdatePayload, toAppend)
			}
			nodes.Sender = coreStatusReal.Me.String()
			nodes.Ej = coreStatusReal.Ej
		} else {
			return errors.New("error: bad message passed (incompatible support)")
		}
	default:
		return errors.New("error: got bad state from core")
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	time, err := ntptime.GetNTPTime()
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}
	nodes.Timestamp = time.UnixNano()

	_, err = vgc.client.PushGossip(timeout, nodes)

	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	return nil
}
