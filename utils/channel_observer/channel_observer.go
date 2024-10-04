package gossip

import (
	lockedmap "github.com/sebastianopriscan/GNCFD/utils/locked_map"
)

//Subject implementation

type ChannelObserverSubject interface {
	RegisterChannel(chan any) int
	UnregisterChannel(int) bool
	PushToChannels(any)
}

type ChannelObserverSubjectImpl struct {
	channels lockedmap.LockedMap[int, chan any]

	freeentries int
}

func NewChannelObserverSubjectImpl() ChannelObserverSubjectImpl {
	return ChannelObserverSubjectImpl{
		channels:    lockedmap.LockedMap[int, chan any]{Map: make(map[int]chan any)},
		freeentries: 0,
	}
}

func (chimpl *ChannelObserverSubjectImpl) PushToChannels(nodes any) {

	go func() {
		chimpl.channels.Mu.RLock()
		defer chimpl.channels.Mu.RUnlock()

		for _, channel := range chimpl.channels.Map {
			channel <- nodes
		}
	}()
}

func (chimpl *ChannelObserverSubjectImpl) RegisterChannel(chann chan any) int {
	chimpl.channels.Mu.Lock()
	defer chimpl.channels.Mu.Unlock()

	if len(chimpl.channels.Map) == 0 {
		chimpl.channels.Map[0] = chann
		chimpl.freeentries = 0
		return 0
	}

	if chimpl.freeentries == 0 {
		chimpl.channels.Map[len(chimpl.channels.Map)] = chann
		return len(chimpl.channels.Map) - 1
	}

	for count := 0; ; count++ {
		if _, ok := chimpl.channels.Map[count]; !ok {
			chimpl.channels.Map[count] = chann
			chimpl.freeentries--
			return count
		}
	}
}

func (chimpl *ChannelObserverSubjectImpl) UnregisterChannel(num int) bool {
	chimpl.channels.Mu.Lock()
	defer chimpl.channels.Mu.Unlock()

	chann, ok := chimpl.channels.Map[num]

	if ok {
		close(chann)
		if num != len(chimpl.channels.Map)-1 {
			chimpl.freeentries++
		}
		delete(chimpl.channels.Map, num)
	}

	return false
}

//Observer implementation

type Chancode struct {
	Chann chan any
	Code  int
}

type ChannelObserverObserver struct {
	Registrations lockedmap.LockedMap[ChannelObserverSubject, Chancode]
}

func (obs *ChannelObserverObserver) ObserveSubject(subj ChannelObserverSubject) {
	chann := make(chan any, 10)
	code := subj.RegisterChannel(chann)
	obs.Registrations.Mu.Lock()
	defer obs.Registrations.Mu.Unlock()
	obs.Registrations.Map[subj] = Chancode{Chann: chann, Code: code}
}

func (obs *ChannelObserverObserver) UnfollowSubject(subj ChannelObserverSubject) bool {
	obs.Registrations.Mu.Lock()
	defer obs.Registrations.Mu.Unlock()
	entry, ok := obs.Registrations.Map[subj]

	if !ok {
		return false
	}

	return subj.UnregisterChannel(entry.Code)
}
