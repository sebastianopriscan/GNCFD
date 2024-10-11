package endpoints

import (
	"fmt"

	"github.com/sebastianopriscan/GNCFD/core"
	connectionmanager "github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/connection_manager"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
	channelobserver "github.com/sebastianopriscan/GNCFD/utils/channel_observer"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
	lockedmap "github.com/sebastianopriscan/GNCFD/utils/locked_map"
	"google.golang.org/grpc"
)

type VivaldiGRPCServerDesc struct {
	Server  *connectionmanager.ServerInterface
	Exists  bool
	VivServ *VivaldiGRPCGossipServer
}

func ActivateVivaldiGRPCServer(name string, addr string, transport string,
	opts []grpc.ServerOption, coreMap *lockedmap.LockedMap[guid.Guid, core.GNCFDCoreInteractionGate]) (*VivaldiGRPCServerDesc, error) {

	serv, exist, err := connectionmanager.GetServer(name, addr, transport, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving server, details: %s", err)
	}

	vivserv := &VivaldiGRPCGossipServer{
		coreMap:                    coreMap,
		ChannelObserverSubjectImpl: channelobserver.NewChannelObserverSubjectImpl(),
	}

	pb_go.RegisterGossipStatusServer(serv.Server, vivserv)
	serv.Start()

	return &VivaldiGRPCServerDesc{Server: serv, Exists: exist, VivServ: vivserv}, nil
}

func DeactivateVivaldiGRPCServer(servDesc *VivaldiGRPCServerDesc) error {
	return connectionmanager.ReleaseServerUsage(servDesc.Server)
}
