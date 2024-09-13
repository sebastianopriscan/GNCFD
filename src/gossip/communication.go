package gossip

import (
	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/core/guid"
)

type CommunicationChannel interface {
	Push(nodeCore core.GNCFDCore, coreData core.Metadata, messageID guid.Guid) error
	Pull(nodeCore core.GNCFDCore) error
	Exchange(nodeCore core.GNCFDCore, coreData core.Metadata, messageID guid.Guid) error
	Forward(nodeCore core.GNCFDCore, data core.Metadata) error
}
