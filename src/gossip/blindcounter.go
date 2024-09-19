package gossip

import (
	"fmt"
	"log"

	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
)

type MessageToForward struct {
	MessageID guid.Guid
	Sender    guid.Guid
	Payload   any
}

type messageHistory struct {
	patience           int
	already_sent_peers map[guid.Guid]guid.Guid
}

type BlindCounterGossiper struct {
	ChannelObserverObserver

	peers *LockedMap[guid.Guid, CommunicationChannel]
	core  core.GNCFDCore

	B int
	F int

	inputchann chan bool

	history map[guid.Guid]*messageHistory

	stopchann chan bool
}

func NewBlindCounterGossiper(peerMap *LockedMap[guid.Guid, CommunicationChannel], core core.GNCFDCore, B int, F int) *BlindCounterGossiper {
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

	msg_history.already_sent_peers[forwdMsg.Sender] = forwdMsg.Sender

	b_neighbors := make([]guid.Guid, bcg.B)
	b_neigh_idx := 0

	bcg.peers.Mu.RLock()
	for neigh := range bcg.peers.Map {
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
		err := bcg.peers.Map[b_neighbors[i]].Forward(bcg.core, forwdMsg.Payload)
		if err != nil {
			failedPeers = append(failedPeers, b_neighbors[i])
		} else {
			msg_history.already_sent_peers[b_neighbors[i]] = b_neighbors[i]
		}
	}
	bcg.peers.Mu.RUnlock()

	bcg.core.SignalFailed(failedPeers)

	msg_history.patience--
}

func do_gossip_push(bcg *BlindCounterGossiper) error {

	messageID, err := guid.GenerateGUID()
	if err != nil {
		return fmt.Errorf("error generating message guid, details: %s", err)
	}

	bcg.history[messageID] = &messageHistory{patience: bcg.F, already_sent_peers: make(map[guid.Guid]guid.Guid)}
	msg_history := bcg.history[messageID]

	b_neighbors := make([]guid.Guid, bcg.B)
	b_neigh_idx := 0

	bcg.peers.Mu.RLock()
	for neigh := range bcg.peers.Map {
		if _, present := msg_history.already_sent_peers[neigh]; !present {
			b_neighbors[b_neigh_idx] = neigh
			b_neigh_idx++
			if b_neigh_idx == bcg.B {
				break
			}
		}
	}

	updates, err := bcg.core.GetStateUpdates()
	if err != nil {
		return fmt.Errorf("error getting core updates for pushing, details: %s", err)
	}

	failedPeers := make([]guid.Guid, 0, bcg.B)
	for i := 0; i < b_neigh_idx; i++ {
		err := bcg.peers.Map[b_neighbors[i]].Push(bcg.core, updates, messageID)
		if err != nil {
			failedPeers = append(failedPeers, b_neighbors[i])
		} else {
			msg_history.already_sent_peers[b_neighbors[i]] = b_neighbors[i]
		}
	}
	bcg.peers.Mu.RUnlock()

	bcg.core.SignalFailed(failedPeers)

	msg_history.patience--

	return nil
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
			do_gossip_push(bcg)

		default:
			for _, v := range bcg.Registrations {
				select {
				case data := <-v.chann:

					switch mssg := data.(type) {
					case *MessageToForward:
						msg_history, ok := bcg.history[mssg.MessageID]
						if !ok {
							bcg.history[mssg.MessageID] = &messageHistory{patience: bcg.F,
								already_sent_peers: make(map[guid.Guid]guid.Guid, 0)}

							msg_history = bcg.history[mssg.MessageID]
						}

						if msg_history.patience <= 0 {

							if msg_history.patience == -100 {
								delete(bcg.history, mssg.MessageID)
							} else {
								msg_history.patience--
							}
							continue
						}

						do_gossip_forward(bcg, msg_history, mssg)
					case bool:
						if err := do_gossip_push(bcg); err != nil {
							log.Printf("error pushing new gossip, details: %s\n", err)
						}
					default:
						log.Println("message passed in bad format, skipping")
						continue
					}
				default:
					continue
				}
			}
		}
	}
}
