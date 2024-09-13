package gossip

import (
	"github.com/sebastianopriscan/GNCFD/core"
)

type CommunicationChannel interface {
	Push(nodeCore core.GNCFDCore) error
	Pull(nodeCore core.GNCFDCore) error
	Exchange(nodeCore core.GNCFDCore) error
	Forward(data core.Metadata) error
}
