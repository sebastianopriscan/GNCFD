package gossip

import (
	"sync"
)

type LockedMap[K comparable, V any] struct {
	Mu  sync.RWMutex
	Map map[K]V
}
