package endpoints

import (
	"fmt"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/gossip"
	connectionmanager "github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/connection_manager"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
	"google.golang.org/grpc"
)

type VivaldiGRPCServerDesc struct {
	Server  *connectionmanager.ServerInterface
	Exists  bool
	VivServ *VivaldiGRPCGossipServer
}

func ActivateVivaldiGRPCServer(name string, addr string, transport string,
	opts []grpc.ServerOption, coreMap *gossip.LockedMap[guid.Guid, core.GNCFDCore]) (*VivaldiGRPCServerDesc, error) {

	serv, exist, err := connectionmanager.GetServer(name, addr, transport, opts)
	if err != nil {
		return nil, fmt.Errorf("error retrieving server, details: %s", err)
	}

	vivserv := &VivaldiGRPCGossipServer{
		coreMap:                    coreMap,
		ChannelObserverSubjectImpl: gossip.NewChannelObserverSubjectImpl(),
	}

	pb_go.RegisterGossipStatusServer(serv.Server, vivserv)

	return &VivaldiGRPCServerDesc{Server: serv, Exists: exist, VivServ: vivserv}, nil
}

func DeactivateVivaldiGRPCServer(servDesc *VivaldiGRPCServerDesc) error {
	return connectionmanager.ReleaseServerUsage(servDesc.Server)
}
