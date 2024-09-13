package gossip

import (
	"sync"

	"github.com/sebastianopriscan/GNCFD/core/guid"
)

type LockedCommunicaitonMap struct {
	mu    sync.Mutex
	peers map[guid.Guid]CommunicationChannel
}
