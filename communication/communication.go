package communication

import (
	"github.com/sebastianopriscan/GNCFD/core"
	"github.com/sebastianopriscan/GNCFD/utils/guid"
)

type GNCFDCommunicationChannel interface {
	Push(nodeCore core.GNCFDCoreInteractionGate, coreData core.CoreData, messageID guid.Guid) error
	Pull(nodeCore core.GNCFDCoreInteractionGate) error
	Exchange(nodeCore core.GNCFDCoreInteractionGate, coreData core.CoreData, messageID guid.Guid) error
	Forward(nodeCore core.GNCFDCoreInteractionGate, data core.CoreData) error
}
