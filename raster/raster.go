package raster

import (
	"image/color"

	"golang.org/x/mobile/geom"
)

type Path []geom.Pt

type Shape interface {
	Path() Path
}

type Circle struct {
	Center geom.Point
	Radius geom.Pt
}

type Ellipse struct {
	Center geom.Point
	Radius geom.Point
}

type Cap int8

const (
	ButtCap Cap = iota
	RoundCap
	SquareCap
)

type Join int8

const (
	Bevel Join = iota
	Round
)

type Drawable struct {
	Shape  Shape
	Fill   color.Color
	Stroke struct {
		Width geom.Pt
		Cap   Cap
		Join  Join
		Color color.Color
	}
}
