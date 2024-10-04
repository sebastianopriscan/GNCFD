package core

import (
	"github.com/sebastianopriscan/GNCFD/utils/guid"
)

type Metadata interface {
}

type GNCFDCore interface {
	GetClosestOf(guids []guid.Guid) ([]guid.Guid, error)
	GetIsFailed(guid guid.Guid) bool

	GetCoreSession() guid.Guid
	SetCoreSession(guid.Guid)

	GetKind() string
	GetStateUpdates() (Metadata, error)
	GetMyState() (Metadata, error)
	UpdateState(metadata Metadata) error

	SignalFailed(peers []guid.Guid)
}
