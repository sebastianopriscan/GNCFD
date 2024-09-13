package gossip

import "sync"

//Subject implementation

type ChannelObserverSubject interface {
	RegisterChannel(chan any) int
	UnregisterChannel(int) bool
	PushToChannels(any)
}

type ChannelObserverSubjectImpl struct {
	channels_mu sync.Mutex
	channels    map[int]chan any

	freeentries int
}

func (chimpl *ChannelObserverSubjectImpl) PushToChannels(nodes any) {

	go func() {
		chimpl.channels_mu.Lock()
		defer chimpl.channels_mu.Unlock()

		for _, channel := range chimpl.channels {
			channel <- nodes
		}
	}()
}

func (chimpl *ChannelObserverSubjectImpl) RegisterChannel(chann chan any) int {
	chimpl.channels_mu.Lock()
	defer chimpl.channels_mu.Unlock()

	if len(chimpl.channels) == 0 {
		chimpl.channels[0] = chann
		chimpl.freeentries = 0
		return 0
	}

	if chimpl.freeentries == 0 {
		chimpl.channels[len(chimpl.channels)] = chann
		return len(chimpl.channels) - 1
	}

	for count := 0; ; count++ {
		if _, ok := chimpl.channels[count]; !ok {
			chimpl.channels[count] = chann
			chimpl.freeentries--
			return count
		}
	}
}

func (chimpl *ChannelObserverSubjectImpl) UnregisterChannel(num int) bool {
	chimpl.channels_mu.Lock()
	defer chimpl.channels_mu.Unlock()

	chann, ok := chimpl.channels[num]

	if ok {
		close(chann)
		if num != len(chimpl.channels)-1 {
			chimpl.freeentries++
		}
		delete(chimpl.channels, num)
	}

	return false
}

//Observer implementation

type chancode struct {
	chann chan any
	code  int
}

type ChannelObserverObserver struct {
	Registrations map[ChannelObserverSubject]chancode
}

func (obs *ChannelObserverObserver) ObserveSubject(subj ChannelObserverSubject) {
	chann := make(chan any, 10)
	code := subj.RegisterChannel(chann)
	obs.Registrations[subj] = chancode{chann: chann, code: code}
}

func (obs *ChannelObserverObserver) UnfollowSubject(subj ChannelObserverSubject) bool {
	entry, ok := obs.Registrations[subj]

	if !ok {
		return false
	}

	return subj.UnregisterChannel(entry.code)
}
