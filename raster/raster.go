// Package raster is experimental vector drawing for the sprite package.
//
// WARNING: likely to change.
// This is an experiment in APIs. Implemented using freetype/raster for now.
package raster

import (
	"fmt"
	"image"
	"image/color"
	"math"

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
	return ftraster.Fix32(float32(p) * geom.PixelsPerPt * 255)
}

func fix32ToPt(p ftraster.Fix32) geom.Pt {
	return geom.Pt((float32(p>>8) + float32(p&0xff)/0xff) / geom.PixelsPerPt)
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

type Circle struct {
	Center geom.Point
	Radius geom.Pt
}

func (c *Circle) Contains(p geom.Point) bool {
	x := p.X - c.Center.X
	y := p.Y - c.Center.Y
	return x*x+y*y < c.Radius*c.Radius
}

func (c *Circle) Path() (p Path) {
	// No, you cannot draw a circle with Bezier curves.
	// But you can do a pretty good approximation.
	//
	// One quadratic bezier for each 45 degree arc.
	// Eight in total for a circle with the endpoints:
	//
	//	     N
	//	  NW   NE
	//	W         E
	//	  SW   SE
	//	     S
	//
	// The cartesian offset of the intercardinal control points
	// is x1 where
	//
	//	cos(Pi/4) = x1 / radius
	//
	// The middle control points of each quadratic bezier arc
	// is the intersection of the tangents of the end points. One
	// end point is always a cardinal direction, the other is an
	// intercardinal. The cartesian offset from the cardinal is x2
	// where
	//
	//	tan(Pi/8) = x2 / radius.
	x1 := geom.Pt(math.Cos(math.Pi/4)) * c.Radius
	x2 := geom.Pt(math.Tan(math.Pi/8)) * c.Radius

	p.AddStart(
		geom.Point{c.Center.X, c.Center.Y - c.Radius}, // N
	)
	p.AddQuadratic(
		geom.Point{c.Center.X + x2, c.Center.Y - c.Radius}, // N-NE
		geom.Point{c.Center.X + x1, c.Center.Y - x1},       // NE
	)
	p.AddQuadratic(
		geom.Point{c.Center.X + c.Radius, c.Center.Y - x2}, // NE-E
		geom.Point{c.Center.X + c.Radius, c.Center.Y},      // E
	)
	p.AddQuadratic(
		geom.Point{c.Center.X + c.Radius, c.Center.Y + x2}, // E-SE
		geom.Point{c.Center.X + x1, c.Center.Y + x1},       // SE
	)
	p.AddQuadratic(
		geom.Point{c.Center.X + x2, c.Center.Y + c.Radius}, // SE-S
		geom.Point{c.Center.X, c.Center.Y + c.Radius},      // S
	)
	p.AddQuadratic(
		geom.Point{c.Center.X - x2, c.Center.Y + c.Radius}, // S-SW
		geom.Point{c.Center.X - x1, c.Center.Y + x1},       // SW
	)
	p.AddQuadratic(
		geom.Point{c.Center.X - c.Radius, c.Center.Y + x2}, // SW-W
		geom.Point{c.Center.X - c.Radius, c.Center.Y},      // W
	)
	p.AddQuadratic(
		geom.Point{c.Center.X - c.Radius, c.Center.Y - x2}, // W-NW
		geom.Point{c.Center.X - x1, c.Center.Y - x1},       // NW
	)
	p.AddQuadratic(
		geom.Point{c.Center.X - x2, c.Center.Y - c.Radius}, // NW-N
		geom.Point{c.Center.X, c.Center.Y - c.Radius},      // N
	)
	return p
}

// TODO: rounded corners (i.e. css border-radius)?
type Rectangle geom.Rectangle

func (r *Rectangle) Path() (p Path) {
	topRight := r.Min
	topRight.X = r.Max.X
	bottomLeft := r.Min
	bottomLeft.Y = r.Max.Y

	p.AddStart(r.Min)
	p.AddLine(topRight)
	p.AddLine(r.Max)
	p.AddLine(bottomLeft)
	p.AddLine(r.Min)
	return p
}

// TODO
type Ellipse struct {
	Center geom.Point
	Radius geom.Point
}

type Drawable struct {
	Shape Shape
	Color color.Color // TODO mask? so many possibilities
}

// Draw is a portable implementation of vector rasterization.
func Draw(dst *image.RGBA, d *Drawable) {
	p := ftraster.NewRGBAPainter(dst)
	p.SetColor(d.Color)
	b := dst.Bounds()
	r := ftraster.NewRasterizer(b.Dx(), b.Dy())
	r.UseNonZeroWinding = true

	pathToFix(r, d.Shape.Path())
	r.Rasterize(p)
}
