package core

import (
	"github.com/sebastianopriscan/GNCFD/utils/guid"
)

type CoreData interface {
}

type GNCFDCore interface {
	GetClosestOf(guids []guid.Guid) ([]guid.Guid, error)
	GetIsFailed(guid guid.Guid) bool
}

type GNCFDCoreInteractionGate interface {
	GetCoreSession() guid.Guid
	SetCoreSession(guid.Guid)

	GetKind() string
	GetStateUpdates() (CoreData, error)
	GetMyState() (CoreData, error)
	UpdateState(metadata CoreData) error

	SignalFailed(peers []guid.Guid)
}
