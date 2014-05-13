// Perlin noise generation code.
//
// Written using
// http://webstaff.itn.liu.se/~stegu/TNM022-2005/perlinnoiselinks/perlin-noise-math-faq.html
// as a guide.
package perlin

import (
	"math"
	"math/rand"
)

type PerlinNoise struct {
	seed   int64
	permut [256]int
	g2d    [256][2]float64 // Randomly generated 2D unit vectors.
}

func NewPerlinNoise(seed int64) *PerlinNoise {
	gen := &PerlinNoise{
		seed: seed,
	}

	// The source's seed is reset to seed for each precomputed set so that any
	// code reordering in this implementation does not alter the noise values
	// produced for a given seed.
	source := rand.NewSource(0)
	rnd := rand.New(source)

	// Initialize gen.permut.
	source.Seed(seed)
	perm := rnd.Perm(len(gen.permut))
	for i := range perm {
		gen.permut[i] = perm[i]
	}

	// Initialize gen.g2d.
	source.Seed(seed)
	for i := range perm {
		randVector(gen.g2d[i][:], rnd)
		normVector(gen.g2d[i][:])
	}

	return gen
}

func (gen *PerlinNoise) grad2d(x, y int) *[2]float64 {
	gradIndex := x&0xff + gen.permut[y&0xff]
	return &gen.g2d[gradIndex&0xff]
}

// At2d returns the noise value at a given 2D point.
func (gen *PerlinNoise) At2d(x, y float64) float64 {
	x0 := floor(x)
	y0 := floor(y)
	x1 := x0 + 1
	y1 := y0 + 1

	// Label corners S=x0,y0, T=x1,y0, U=x0,y1, V=x1,y1.

	gradS := gen.grad2d(int(x0), int(y0))
	gradT := gen.grad2d(int(x1), int(y0))
	gradU := gen.grad2d(int(x0), int(y1))
	gradV := gen.grad2d(int(x1), int(y1))

	// dotX := gradX · ((x,y) - (xX,yX))
	dotS := gradS[0]*(x-x0) + gradS[1]*(y-y0)
	dotT := gradT[0]*(x-x1) + gradT[1]*(y-y0)
	dotU := gradU[0]*(x-x0) + gradU[1]*(y-y1)
	dotV := gradV[0]*(x-x1) + gradV[1]*(y-y1)

	// Bilinear interpolation of the weight between all four points, but using an
	// "ease" function.
	dx := x - x0
	sx := 3*dx*dx - 2*dx*dx*dx
	a := dotS + sx*(dotT-dotS)
	b := dotU + sx*(dotV-dotU)

	dy := y - y0
	sy := 3*dy*dy - 2*dy*dy*dy

	return a + sy*(b-a)
}

func (gen *PerlinNoise) MeanMagnitude() float64 {
	return 0.5
}

// Utility functions follow.

// floor is equivalent to math.Floor, but faster - fails outside the int32
// range.
func floor(n float64) float64 {
	if n >= 0 {
		return float64(int32(n))
	}
	return float64(int32(n) - 1)
}

// randVector generates a random vector whose components are each in the range
// [-1, 1). The dimensionality of the vector is len(c).
func randVector(c []float64, rnd *rand.Rand) {
	for i := range c {
		c[i] = 2*rnd.Float64() - 1
	}
}

// normVector normalizes the passed vector. The dimensionality of the vector is
// len(c).
func normVector(c []float64) {
	magnitude := float64(0)
	for _, v := range c {
		magnitude += math.Pow(v, 2)
	}
	length := math.Sqrt(magnitude)
	for i := range c {
		c[i] /= length
	}
}
