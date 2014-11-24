// Package raster is experimental vector drawing for the sprite package.
//
// WARNING: likely to change.
// This is an experiment in APIs. Implemented using freetype/raster for now.
package raster

import (
	"fmt"
	"image"
	"image/color"

	ftraster "code.google.com/p/freetype-go/freetype/raster"
	"golang.org/x/mobile/geom"
)

// A Path is a sequence of curves.
type Path []geom.Pt

func (p *Path) Path() Path {
	return *p
}

// AddStart starts a new curve at the given point.
func (p *Path) AddStart(a geom.Point) {
	*p = append(*p, 0, a.X, a.Y, 0)
}

// AddLine adds a linear segment to the current curve.
func (p *Path) AddLine(b geom.Point) {
	*p = append(*p, 1, b.X, b.Y, 1)
}

// AddQuadratic adds a quadratic segment to the current curve.
func (p *Path) AddQuadratic(b, c geom.Point) {
	*p = append(*p, 2, b.X, b.Y, c.X, c.Y, 2)
}

// AddCubic adds a cubic segment to the current curve.
func (p *Path) AddCubic(b, c, d geom.Point) {
	*p = append(*p, 3, b.X, b.Y, c.X, c.Y, d.X, d.Y, 3)
}

type Shape interface {
	Path() Path
}

func ptToFix32(p geom.Pt) ftraster.Fix32 {
	return ftraster.Fix32(float32(p) / geom.PixelsPerPt * 255)
}

func fix32ToPt(p ftraster.Fix32) geom.Pt {
	return geom.Pt(geom.PixelsPerPt * (float32(p>>8) + float32(p&0xff)/0xff))
}

func pathToFix(dst ftraster.Adder, src Path) {
	pt := func(i int) ftraster.Point {
		return ftraster.Point{ptToFix32(src[i]), ptToFix32(src[i+1])}
	}
	i := 0
	for i < len(src) {
		switch src[i] {
		case 0:
			dst.Start(pt(i + 1))
			i += 4
		case 1:
			dst.Add1(pt(i + 1))
			i += 4
		case 2:
			dst.Add2(pt(i+1), pt(i+3))
			i += 6
		case 3:
			dst.Add3(pt(i+1), pt(i+3), pt(i+5))
			i += 8
		default:
			panic(fmt.Sprintf("invalid path, src[%d]=%f", i, src[i]))
		}
	}
}

func fixToPath(src ftraster.Path) Path {
	pt := func(i int) geom.Point {
		return geom.Point{fix32ToPt(src[i]), fix32ToPt(src[i+1])}
	}
	dst := make(Path, 0, len(src))
	i := 0
	for i < len(src) {
		switch src[i] {
		case 0:
			dst.AddStart(pt(i + 1))
			i += 4
		case 1:
			dst.AddLine(pt(i + 1))
			i += 4
		case 2:
			dst.AddQuadratic(pt(i+1), pt(i+3))
			i += 6
		case 3:
			dst.AddCubic(pt(i+1), pt(i+3), pt(i+5))
			i += 8
		default:
			panic(fmt.Sprintf("invalid path, src[%d]=%f", i, src[i]))
		}
	}
	return dst
}

type Capper interface {
	Cap(p *Path, halfWidth geom.Pt, pivot, n1 geom.Point)
}

type capperToRasterCapper struct {
	c Capper
}

func (c *capperToRasterCapper) Cap(p ftraster.Adder, halfWidth ftraster.Fix32, pivot, n1 ftraster.Point) {
	var path Path
	pivotg := fix32PointToPoint(pivot)
	n1g := fix32PointToPoint(n1)
	c.c.Cap(&path, fix32ToPt(halfWidth), pivotg, n1g)
	pathToFix(p, path)
}

type Joiner interface {
	Join(left, right *Path, halfWidth geom.Pt, pivot, n0, n1 geom.Point)
}

type Stroke struct {
	Shape Shape
	Width geom.Pt
	Cap   Capper
	Join  Joiner
}

type joinerToRasterJoiner struct {
	j Joiner
}

func (j *joinerToRasterJoiner) Join(lhs, rhs ftraster.Adder, halfWidth ftraster.Fix32, pivot, n0, n1 ftraster.Point) {
	var left, right Path
	pivotg := fix32PointToPoint(pivot)
	n0g := fix32PointToPoint(n0)
	n1g := fix32PointToPoint(n1)
	j.j.Join(&left, &right, fix32ToPt(halfWidth), pivotg, n0g, n1g)
	pathToFix(lhs, left)
	pathToFix(rhs, right)
}

func fix32PointToPoint(p ftraster.Point) geom.Point {
	return geom.Point{fix32ToPt(p.X), fix32ToPt(p.Y)}
}

func (s *Stroke) Path() Path {
	// TODO: implement Stroke directly on geom.Pt?
	dst := ftraster.Path{}
	srcFix := s.Shape.Path()
	src := make(ftraster.Path, 0, len(srcFix))
	pathToFix(&src, srcFix)

	var cr ftraster.Capper
	var jr ftraster.Joiner
	if s.Cap != nil {
		cr = &capperToRasterCapper{s.Cap}
	}
	if s.Join != nil {
		jr = &joinerToRasterJoiner{s.Join}
	}
	ftraster.Stroke(&dst, src, ptToFix32(s.Width), cr, jr)
	return fixToPath(dst)
}

// TODO
type Circle struct {
	Center geom.Point
	Radius geom.Pt
}

// TODO
type Ellipse struct {
	Center geom.Point
	Radius geom.Point
}

type Drawable struct {
	Shape Shape
	// TODO color? mask? so many things.
}

// Draw is a portable implementation of vector rasterization.
func Draw(dst *image.RGBA, d *Drawable) {
	p := ftraster.NewRGBAPainter(dst)
	p.SetColor(color.RGBA{0xff, 0, 0, 0xff})
	b := dst.Bounds()
	r := ftraster.NewRasterizer(b.Dx(), b.Dy())

	pathToFix(r, d.Shape.Path())
	r.Rasterize(p)
}
