package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	. "github.com/huin/chunkymonkey/generation"
	"github.com/huin/chunkymonkey/perlin"
)

type Stat struct {
	count    int
	min, max float64
}

func (s *Stat) Add(sample float64) {
	if s.count == 0 {
		s.max = sample
		s.min = sample
	} else {
		if sample > s.max {
			s.max = sample
		}
		if sample < s.min {
			s.min = sample
		}
	}
	s.count++
}

func main() {
	w := 512
	h := 512

	perlin := perlin.NewPerlinNoise(0)

	gen := &Sum{
		Inputs: []ISource{
			&Turbulence{
				Dx:     &Scale{50, 1, &Offset{20.1, 0, perlin}},
				Dy:     &Scale{50, 1, &Offset{10.1, 0, perlin}},
				Factor: 100,
				Source: &Scale{
					Wavelength: 200,
					Amplitude:  100,
					Source:     perlin,
				},
			},
			&Turbulence{
				Dx:     &Scale{40, 1, &Offset{20.1, 0, perlin}},
				Dy:     &Scale{40, 1, &Offset{10.1, 0, perlin}},
				Factor: 10,
				Source: &Mult{
					A: &Scale{
						Wavelength: 40,
						Amplitude:  20,
						Source:     perlin,
					},
					// Local steepness.
					B: &Scale{
						Wavelength: 200,
						Amplitude:  1,
						Source:     &Add{perlin, 0.6},
					},
				},
			},
			&Scale{
				Wavelength: 5,
				Amplitude:  2,
				Source:     perlin,
			},
		},
	}

	values := make([]float64, h*h)

	var valueStat Stat

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			value := gen.At2d(float64(col), float64(row))
			valueStat.Add(value)
			values[row*w+col] = value
		}
	}

	img := image.NewGray(image.Rect(0, 0, w, h))

	base := valueStat.min
	scale := 255 / (valueStat.max - valueStat.min)

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			scaled := scale * (values[row*w+col] - base)
			img.Set(col, row, color.Gray{uint8(scaled)})
		}
	}

	log.Printf("value stats %#v", valueStat)

	outFile, err := os.Create("output.png")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()
	png.Encode(outFile, img)
}
