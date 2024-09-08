package connectionmanager

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sebastianopriscan/GNCFD/core/guid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type connCount struct {
	conn  *grpc.ClientConn
	count int
}

var mu sync.Mutex

var openCommunications map[guid.Guid]connCount = make(map[guid.Guid]connCount)

type GrpcCommunicationChannel struct {
	Peer    guid.Guid
	Address string
	Conn    *grpc.ClientConn
	set     bool
}

func NewGrpcCommunicationChannel(peer guid.Guid, address string) (*GrpcCommunicationChannel, error) {

	retVal := &GrpcCommunicationChannel{}

	var conn *grpc.ClientConn

	mu.Lock()

	if connection, ok := openCommunications[peer]; ok {
		conn = connection.conn
		connection.count++
	} else {

		insecure := grpc.WithTransportCredentials(insecure.NewCredentials())

		var err error
		conn, err = grpc.NewClient(address, insecure)
		if err != nil {
			mu.Unlock()
			return nil, fmt.Errorf("unable to create grpc connection, details: %s", err)
		}

		openCommunications[peer] = connCount{conn: conn, count: 1}
	}

	mu.Unlock()

	retVal.Peer = peer
	retVal.Address = address
	retVal.Conn = conn
	retVal.set = true

	return retVal, nil
}

func InvalidateGrpcCommunicationChannel(chann *GrpcCommunicationChannel) error {
	if !chann.set {
		return errors.New("destroy a valid channel")
	}

	mu.Lock()
	defer mu.Unlock()

	cnct := openCommunications[chann.Peer]

	if cnct.count == 1 {
		delete(openCommunications, chann.Peer)
	} else {
		cnct.count--
	}

	chann.set = false

	return nil
}
