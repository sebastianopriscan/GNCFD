package gossip

import (
	"fmt"
	"log"
	"time"

	"github.com/sebastianopriscan/GNCFD/core"
	channelobserver "github.com/sebastianopriscan/GNCFD/utils/channel_observer"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
	lockedmap "github.com/sebastianopriscan/GNCFD/utils/locked_map"
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
	channelobserver.ChannelObserverObserver

	peers *lockedmap.LockedMap[guid.Guid, GNCFDCommunicationChannel]
	core  core.GNCFDCoreInteractionGate

	B int
	F int

	inputchann chan bool

	history lockedmap.LockedMap[guid.Guid, *messageHistory]

	stopchann chan bool
}

func NewBlindCounterGossiper(peerMap *lockedmap.LockedMap[guid.Guid, GNCFDCommunicationChannel], core core.GNCFDCoreInteractionGate, B int, F int) *BlindCounterGossiper {
	retVal := &BlindCounterGossiper{peers: peerMap,
		core: core, B: B, F: F,

		inputchann: make(chan bool, 10),
		history:    lockedmap.LockedMap[guid.Guid, *messageHistory]{Map: make(map[guid.Guid]*messageHistory)},
		ChannelObserverObserver: channelobserver.ChannelObserverObserver{
			Registrations: lockedmap.LockedMap[channelobserver.ChannelObserverSubject, channelobserver.Chancode]{
				Map: make(map[channelobserver.ChannelObserverSubject]channelobserver.Chancode),
			},
		},
	}

	asSubject, ok := core.(channelobserver.ChannelObserverSubject)
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

	bcg.history.Mu.Lock()
	defer bcg.history.Mu.Unlock()

	bcg.history.Map[messageID] = &messageHistory{patience: bcg.F, already_sent_peers: make(map[guid.Guid]guid.Guid)}
	msg_history := bcg.history.Map[messageID]

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

	cleaner_stopchann := make(chan bool)
	go bcg.message_history_cleaner(&cleaner_stopchann)

	for {
		select {
		case <-bcg.stopchann:

			cleaner_stopchann <- true
			close(cleaner_stopchann)

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
			bcg.Registrations.Mu.RLock()
			for _, v := range bcg.Registrations.Map {
				select {
				case data := <-v.Chann:

					switch mssg := data.(type) {
					case *MessageToForward:

						bcg.history.Mu.Lock()

						msg_history, ok := bcg.history.Map[mssg.MessageID]
						if !ok {
							bcg.history.Map[mssg.MessageID] = &messageHistory{patience: bcg.F,
								already_sent_peers: make(map[guid.Guid]guid.Guid, 0)}

							msg_history = bcg.history.Map[mssg.MessageID]
						}

						if msg_history.patience == 0 {
							continue
						}

						do_gossip_forward(bcg, msg_history, mssg)

						bcg.history.Mu.Unlock()
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
			bcg.Registrations.Mu.RUnlock()
		}
	}
}

func (bcg *BlindCounterGossiper) message_history_cleaner(stopchann *chan bool) {

	for {
		select {
		case <-*stopchann:
			return
		default:
			time.Sleep(10 * time.Second)
			bcg.history.Mu.Lock()

			for k, v := range bcg.history.Map {
				if v.patience == 0 {
					delete(bcg.history.Map, k)
				}
			}

			bcg.history.Mu.Unlock()
		}
	}
}
