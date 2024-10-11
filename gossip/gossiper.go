package gossip

import (
	"github.com/sebastianopriscan/GNCFD/communication"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
)

type GNCFDGossiper interface {
	StartGossiping() bool
	StopGossiping()
	InsertGossip() bool

	AddPeer(guid.Guid, communication.GNCFDCommunicationChannel)
	RemovePeer(guid.Guid)
}
