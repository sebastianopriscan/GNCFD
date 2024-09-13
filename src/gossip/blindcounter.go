package gossip

import (
	"log"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
)

type MessageToForward struct {
	messageID guid.Guid
	sender    guid.Guid
	payload   any
}

type messageHistory struct {
	patience           int
	already_sent_peers map[guid.Guid]guid.Guid
}

type BlindCounterGossiper struct {
	ChannelObserverObserver

	peers *LockedCommunicaitonMap
	core  core.GNCFDCore

	B int
	F int

	inputchann chan bool

	history map[guid.Guid]*messageHistory

	stopchann chan bool
}

func NewBlindCounterGossiper(peerMap *LockedCommunicaitonMap, core core.GNCFDCore, B int, F int) *BlindCounterGossiper {
	retVal := &BlindCounterGossiper{peers: peerMap,
		core: core, B: B, F: F,

		inputchann:              make(chan bool, 10),
		history:                 make(map[guid.Guid]*messageHistory),
		ChannelObserverObserver: ChannelObserverObserver{make(map[ChannelObserverSubject]chancode)},
	}

	asSubject, ok := core.(ChannelObserverSubject)
	if !ok {
		log.Println("core was not a subject: only direct communication with gossiper supported")
	} else {
		retVal.ObserveSubject(asSubject)
	}

	return retVal
}

func (bgc *BlindCounterGossiper) StartGossiping() bool {

	if bgc.stopchann != nil {
		return false
	}

	bgc.stopchann = make(chan bool)
	go bgc.gossip_routine()

	return true
}

func (bgc *BlindCounterGossiper) StopGossiping() {

	if bgc.stopchann == nil {
		return
	}

	bgc.stopchann <- true
	close(bgc.stopchann)
}

func (bgc *BlindCounterGossiper) InsertGossip() bool {
	if bgc.stopchann == nil {
		return false
	}
	bgc.inputchann <- true
	return true
}

func do_gossip_forward(bcg *BlindCounterGossiper, msg_history *messageHistory, forwdMsg *MessageToForward) {

	msg_history.already_sent_peers[forwdMsg.sender] = forwdMsg.sender

	b_neighbors := make([]guid.Guid, bcg.B)
	b_neigh_idx := 0

	bcg.peers.mu.Lock()
	for neigh := range bcg.peers.peers {
		if _, present := msg_history.already_sent_peers[neigh]; !present {
			b_neighbors[b_neigh_idx] = neigh
			b_neigh_idx++
			if b_neigh_idx == bcg.B {
				break
			}
		}
	}

	failedPeers := make([]guid.Guid, 0, bcg.B)
	for i := 0; i < b_neigh_idx; i++ {
		err := bcg.peers.peers[b_neighbors[i]].Forward(forwdMsg.payload)
		if err != nil {
			failedPeers = append(failedPeers, b_neighbors[i])
		} else {
			msg_history.already_sent_peers[b_neighbors[i]] = b_neighbors[i]
		}
	}
	bcg.peers.mu.Unlock()

	bcg.core.SignalFailed(failedPeers)

	msg_history.patience--
}

func (bcg *BlindCounterGossiper) gossip_routine() {

	for {
		select {
		case <-bcg.stopchann:
			for {
				_, ok := <-bcg.stopchann
				if !ok {
					bcg.stopchann = nil
					return
				}
			}

		case <-bcg.inputchann:
			data, err := bcg.core.GetStateUpdates()
			if err != nil {
				log.Printf("unable to get core updates, details: %s", err)
				continue
			}

			forwdMsg, ok := data.(MessageToForward)
			if !ok {
				log.Printf("message passed in bad format, skipping")
				continue
			}

			bcg.history[forwdMsg.messageID] = &messageHistory{patience: bcg.F,
				already_sent_peers: make(map[guid.Guid]guid.Guid)}

			do_gossip_forward(bcg, bcg.history[forwdMsg.messageID], &forwdMsg)

		default:
			for _, v := range bcg.Registrations {
				select {
				case data := <-v.chann:
					forwdMsg, ok := data.(MessageToForward)
					if !ok {
						log.Printf("message passed in bad format, skipping")
						continue
					}

					msg_history, ok := bcg.history[forwdMsg.messageID]
					if !ok {
						bcg.history[forwdMsg.messageID] = &messageHistory{patience: bcg.F,
							already_sent_peers: make(map[guid.Guid]guid.Guid, 0)}

						msg_history = bcg.history[forwdMsg.messageID]
					}

					if msg_history.patience <= 0 {

						if msg_history.patience == -100 {
							delete(bcg.history, forwdMsg.messageID)
						} else {
							msg_history.patience--
						}
						continue
					}

					do_gossip_forward(bcg, msg_history, &forwdMsg)

				default:
					continue
				}
			}
		}
	}
}
