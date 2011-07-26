package generation

import (
	"math"
)

type ISource interface {
	At2d(x, y float64) float64
}

type Const float64

func (gen Const) At2d(x, y float64) float64 {
	return float64(gen)
}

type Offset struct {
	Dx, Dy float64
	Source ISource
}

func (gen *Offset) At2d(x, y float64) float64 {
	return gen.Source.At2d(x+gen.Dx, y+gen.Dy)
}

type Turbulence struct {
	Dx, Dy ISource
	Factor float64
	Source ISource
}

func (gen *Turbulence) At2d(x, y float64) float64 {
	dx := gen.Dx.At2d(x, y) * gen.Factor
	dy := gen.Dy.At2d(x, y) * gen.Factor
	return gen.Source.At2d(x+dx, y+dy)
}

type Scale struct {
	Wavelength float64
	Amplitude  float64
	Source     ISource
}

func (gen *Scale) At2d(x, y float64) float64 {
	return gen.Source.At2d(x/gen.Wavelength, y/gen.Wavelength) * gen.Amplitude
}

type Pow struct {
	Source ISource
	Power  ISource
}

func (gen *Pow) At2d(x, y float64) float64 {
	v := gen.Source.At2d(x, y)
	power := gen.Power.At2d(x, y)
	if v < 0 {
		// Retain sign, and don't break on negatives raised to fractional powers.
		// Not strictly a power function, but serves the purpose.
		return -math.Pow(-v, power)
	}
	return math.Pow(v, power)
}

type Mult struct {
	A, B ISource
}

func (gen *Mult) At2d(x, y float64) float64 {
	return gen.A.At2d(x, y) * gen.B.At2d(x, y)
}

type Add struct {
	Source ISource
	Value  float64
}

func (gen *Add) At2d(x, y float64) float64 {
	return gen.Source.At2d(x, y) + gen.Value
}

type Sum struct {
	Inputs []ISource
}

func (gen *Sum) At2d(x, y float64) float64 {
	var accum float64
	for _, input := range gen.Inputs {
		accum += input.At2d(x, y)
	}
	return accum
}
