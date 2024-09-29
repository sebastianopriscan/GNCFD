package connectionmanager

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type servCount struct {
	addr   string
	server *grpc.Server
	count  int
}

var server_mu sync.Mutex

var availableInterfaces map[string]servCount = make(map[string]servCount)

type ServerInterface struct {
	name    string
	Server  *grpc.Server
	Conn    net.Listener
	Address string
	set     bool
}

func (srv *ServerInterface) Start() {
	go srv.Server.Serve(srv.Conn)
}

func GetServer(name string, addr string, transport string, opts []grpc.ServerOption) (*ServerInterface, bool, error) {

	retVal := &ServerInterface{}

	var server *grpc.Server

	server_mu.Lock()
	var ok bool

	if entry, ok := availableInterfaces[name]; ok {
		server = entry.server
		entry.count++
	} else {

		lis, err := net.Listen(transport, addr)
		if err != nil {
			server_mu.Unlock()
			return nil, false, fmt.Errorf("error: unable to create interface, details: %s", err)
		}

		server = grpc.NewServer(opts...)
		retVal.Conn = lis
		availableInterfaces[name] = servCount{addr: addr, server: server, count: 1}
	}

	server_mu.Unlock()

	retVal.Address = addr
	retVal.Server = server
	retVal.name = name
	retVal.set = true

	return retVal, ok, nil
}

func ReleaseServerUsage(interf *ServerInterface) error {

	if !interf.set {
		return errors.New("release a valid interface")
	}

	server_mu.Lock()
	defer server_mu.Unlock()

	intCt := availableInterfaces[interf.name]

	intCt.count--

	interf.set = false

	return nil
}

func DestroyServer(name string) (bool, error) {

	server_mu.Lock()
	defer server_mu.Unlock()

	intCt, ok := availableInterfaces[name]
	if !ok {
		return false, errors.New("error: server with name not exists")
	}

	if intCt.count != 0 {
		return false, nil
	}

	intCt.server.GracefulStop()
	delete(availableInterfaces, name)

	return true, nil
}
