package gossip

import (
	"sync"

	"github.com/sebastianopriscan/GNCFD/core/guid"
)

type LockedCommunicaitonMap struct {
	Mu    sync.Mutex
	Peers map[guid.Guid]CommunicationChannel
}
