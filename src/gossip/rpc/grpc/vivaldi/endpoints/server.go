package endpoints

import (
	"fmt"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
	connectionmanager "github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/connection_manager"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
	"google.golang.org/grpc"
)

func ActivateVivaldiGRPCServer(name string, addr string, transport string,
	opts []grpc.ServerOption, coreMap map[guid.Guid]core.GNCFDCore) (*connectionmanager.ServerInterface, bool, error) {

	serv, exist, err := connectionmanager.GetServer(name, addr, transport, opts)
	if err != nil {
		return nil, false, fmt.Errorf("error retrieving server, details: %s", err)
	}

	pb_go.RegisterGossipStatusServer(serv.Server, &VivaldiGRPCGossipServer{coreMap: coreMap})

	return serv, exist, nil
}

func DeactivateVivaldiGRPCServer(servImpl *connectionmanager.ServerInterface) error {
	return connectionmanager.ReleaseServerUsage(servImpl)
}
