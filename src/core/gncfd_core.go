package core

import (
	"github.com/sebastianopriscan/GNCFD/core/guid"
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
	UpdateState(metadata Metadata) error

	GetCallback() func(rtt float64, metadata Metadata)
}
