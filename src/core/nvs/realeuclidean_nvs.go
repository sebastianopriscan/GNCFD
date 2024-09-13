package nvs

import (
	"math"
	"math/rand"
)

func euclideanNorm(first []float64, second []float64) float64 {
	sum := 0.
	for i := 0; i < len(first); i++ {
		sum += math.Pow(first[i]-second[i], 2.)
	}

	return math.Sqrt(sum)
}

func euclideanRescale(vector []float64, norm float64) []float64 {
	retVal := make([]float64, len(vector))
	for i, entry := range vector {
		retVal[i] = entry / norm
	}

	return retVal
}

func euclideanExMul(vector []float64, val float64) []float64 {

	retVal := make([]float64, len(vector))
	for i, entry := range vector {
		retVal[i] = entry * val
	}

	return retVal
}

func euclideanRandomEl() float64 {
	return rand.ExpFloat64()
}

func euclideanZero(dim int) []float64 {
	retVal := make([]float64, dim)
	for i := 0; i < dim; i++ {
		retVal[i] = 0.
	}
	return retVal
}

var euclidean_ops = NVSFunctions[float64]{
	Distance:    euclideanNorm,
	Rescaling:   euclideanRescale,
	ExternalMul: euclideanExMul,
	RandomEl:    euclideanRandomEl,
	Zero:        euclideanZero,
}

func NewRealEuclideanSpace(dim int) (*NormedVectorSpace[float64], error) {
	return NewNormedVectorSpace(dim, &euclidean_ops)
}
