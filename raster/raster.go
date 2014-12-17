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
	*p = append(*p, 0, a.X, a.Y)
}

// AddLine adds a linear segment to the current curve.
func (p *Path) AddLine(b geom.Point) {
	*p = append(*p, 1, b.X, b.Y)
}

// AddQuadratic adds a quadratic segment to the current curve.
func (p *Path) AddQuadratic(b, c geom.Point) {
	*p = append(*p, 2, b.X, b.Y, c.X, c.Y)
}

// AddCubic adds a cubic segment to the current curve.
func (p *Path) AddCubic(b, c, d geom.Point) {
	*p = append(*p, 3, b.X, b.Y, c.X, c.Y, d.X, d.Y)
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
			i += 3
		case 1:
			dst.Add1(pt(i + 1))
			i += 3
		case 2:
			dst.Add2(pt(i+1), pt(i+3))
			i += 5
		case 3:
			dst.Add3(pt(i+1), pt(i+3), pt(i+5))
			i += 7
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
			panic(fmt.Sprintf("invalid path, src[%d]=%v", i, src[i]))
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
	//Center geom.Point
	Radius geom.Pt
}

func (c *Circle) Contains(p geom.Point) bool {
	/*
		TODO
		x := p.X - c.Center.X
		y := p.Y - c.Center.Y
		return x*x+y*y < c.Radius*c.Radius
	*/
	return false
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

	// TODO: find a rational foundation for this +1 business.
	cx, cy := c.Radius+1, c.Radius+1

	p.AddStart(
		geom.Point{cx, cy - c.Radius}, // N
	)
	p.AddQuadratic(
		geom.Point{cx + x2, cy - c.Radius}, // N-NE
		geom.Point{cx + x1, cy - x1},       // NE
	)
	p.AddQuadratic(
		geom.Point{cx + c.Radius, cy - x2}, // NE-E
		geom.Point{cx + c.Radius, cy},      // E
	)
	p.AddQuadratic(
		geom.Point{cx + c.Radius, cy + x2}, // E-SE
		geom.Point{cx + x1, cy + x1},       // SE
	)
	p.AddQuadratic(
		geom.Point{cx + x2, cy + c.Radius}, // SE-S
		geom.Point{cx, cy + c.Radius},      // S
	)
	p.AddQuadratic(
		geom.Point{cx - x2, cy + c.Radius}, // S-SW
		geom.Point{cx - x1, cy + x1},       // SW
	)
	p.AddQuadratic(
		geom.Point{cx - c.Radius, cy + x2}, // SW-W
		geom.Point{cx - c.Radius, cy},      // W
	)
	p.AddQuadratic(
		geom.Point{cx - c.Radius, cy - x2}, // W-NW
		geom.Point{cx - x1, cy - x1},       // NW
	)
	p.AddQuadratic(
		geom.Point{cx - x2, cy - c.Radius}, // NW-N
		geom.Point{cx, cy - c.Radius},      // N
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

type Drawable struct {
	Shape Shape
	Color color.Color // TODO mask? so many possibilities
}

type painter struct {
	dst *image.RGBA
	src image.Image
}

func (r *painter) Paint(spans []ftraster.Span, done bool) {
	b := r.dst.Bounds()
	for _, s := range spans {
		if s.Y < b.Min.Y {
			continue
		}
		if s.Y >= b.Max.Y {
			return
		}
		if s.X0 < b.Min.X {
			s.X0 = b.Min.X
		}
		if s.X1 > b.Max.X {
			s.X1 = b.Max.X
		}
		if s.X0 >= s.X1 {
			continue
		}
		// See $GOROOT/pkg/image/draw/draw.go drawCopyOver
		i0 := (s.Y-r.dst.Rect.Min.Y)*r.dst.Stride + (s.X0-r.dst.Rect.Min.X)*4
		i1 := i0 + (s.X1-s.X0)*4
		for i := i0; i < i1; i += 4 {
			sr, sg, sb, sa := r.src.At(s.Y, s.X0+(i-i0)/4).RGBA()

			dr := uint32(r.dst.Pix[i+0])
			dg := uint32(r.dst.Pix[i+1])
			db := uint32(r.dst.Pix[i+2])
			da := uint32(r.dst.Pix[i+3])

			// TODO all wrong
			const m = 1<<16 - 1
			ma := uint32(1)
			ca := uint32(1)
			a := (m - (ca * ma / m)) * 0x101
			r.dst.Pix[i+0] = uint8((dr*a/m + sr) >> 8)
			r.dst.Pix[i+1] = uint8((dg*a/m + sg) >> 8)
			r.dst.Pix[i+2] = uint8((db*a/m + sb) >> 8)
			r.dst.Pix[i+3] = uint8((da*a/m + sa) >> 8)
		}

	}
}

// Draw is a portable implementation of vector rasterization.
func Draw(dst *image.RGBA, src image.Image, path Path) {
	//d *Drawable) {
	p := &painter{dst, src}
	b := dst.Bounds()
	r := ftraster.NewRasterizer(b.Dx(), b.Dy())
	r.UseNonZeroWinding = true

	pathToFix(r, path)
	r.Rasterize(p)
}
