package core

type Metadata interface {
}

type GNCFDCore interface {
	GetClosestOf(guids []int64) ([]int64, error)
	GetIsFailed(guid int64) bool

	GetCallback() func(rtt float64, metadata Metadata)
}
