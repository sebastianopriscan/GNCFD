package endpoints

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/core/impl/vivaldi"
	connectionmanager "github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/connection_manager"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
	"github.com/sebastianopriscan/GNCFD/utils/ntptime"
)

type VivaldiRPCGossipClient struct {
	me     guid.Guid
	client pb_go.GossipStatusClient
	conn   *connectionmanager.GrpcCommunicationChannel
}

func NewVivaldiRPCGossipClient(me guid.Guid, peer guid.Guid, address string) (*VivaldiRPCGossipClient, error) {
	retVal := &VivaldiRPCGossipClient{}

	conn, err := connectionmanager.NewGrpcCommunicationChannel(peer, address)
	if err != nil {
		return nil, fmt.Errorf("error in obtaining connection for client, details: %s", err)
	}

	retVal.client = pb_go.NewGossipStatusClient(conn.Conn)
	retVal.conn = conn
	retVal.me = me

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

func preparePush(nodeCore core.GNCFDCore, updates core.Metadata) (*pb_go.NodeUpdates, error) {

	if nodeCore.GetKind() != core_code {
		return nil, errors.New("error: the requested core is incompatible with this gossip client")
	}

	var pointsToSend pb_go.NodeUpdates
	pointsToSend.CoreSession = nodeCore.GetCoreSession().String()

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

func executePull(nodeCore core.GNCFDCore, nodeUpdates *pb_go.NodeUpdates, time int64) error {

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

func (gc *VivaldiRPCGossipClient) Push(nodeCore core.GNCFDCore, coreData core.Metadata, messageID guid.Guid) error {

	pointsToSend, err := preparePush(nodeCore, coreData)
	if err != nil {
		return fmt.Errorf("error in parameters preparation, details: %s", err)
	}

	pointsToSend.MessageID = messageID.String()
	pointsToSend.Sender = gc.me.String()

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

func (gc *VivaldiRPCGossipClient) Pull(nodeCore core.GNCFDCore) error {

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

func (vgc *VivaldiRPCGossipClient) Exchange(nodeCore core.GNCFDCore, coreData core.Metadata, messageID guid.Guid) error {

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

func (vgc *VivaldiRPCGossipClient) Forward(nodeCore core.GNCFDCore, data core.Metadata) error {

	if nodeCore.GetKind() != core_code {
		return errors.New("error: the requested core is incompatible with this gossip client")
	}

	nodes, ok := data.(*pb_go.NodeUpdates)
	if !ok {
		return errors.New("error: bad message passed")
	}

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := vgc.client.PushGossip(timeout, nodes)

	if err != nil {
		return fmt.Errorf("unable to push state updates, details: %s", err)
	}

	return nil
}
