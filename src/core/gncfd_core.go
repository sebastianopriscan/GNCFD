package core

import (
	"github.com/sebastianopriscan/GNCFD/core/guid"
)

type Metadata interface {
}

type GNCFDCore interface {
	GetClosestOf(guids []guid.Guid) ([]guid.Guid, error)
	GetIsFailed(guid guid.Guid) bool

	GetCallback() func(rtt float64, metadata Metadata)
}
