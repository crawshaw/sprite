package raster

import (
	"fmt"
	"math"

	"golang.org/x/mobile/geom"
)

func (p Path) Bounds() geom.Rectangle {
	r := geom.Rectangle{
		Min: geom.Point{math.MaxFloat32, math.MaxFloat32},
		Max: geom.Point{},
	}
	if len(p) == 0 {
		return r
	}
	include := func(p geom.Point) {
		if p.X < r.Min.X {
			r.Min.X = p.X
		}
		if p.Y < r.Min.Y {
			r.Min.Y = p.Y
		}
		if p.X > r.Max.X {
			r.Max.X = p.X
		}
		if p.Y > r.Max.Y {
			r.Max.Y = p.Y
		}
	}
	var start geom.Point
	for len(p) > 0 {
		switch p[0] {
		case 0, 1:
			end := geom.Point{p[1], p[2]}
			start = end
			p = p[4:]
		case 2:
			n0 := geom.Point{p[1], p[2]}
			end := geom.Point{p[3], p[4]}
			for _, b := range extremitiesQuad(start, n0, end) {
				include(b)
			}
			start = end
			p = p[6:]
		case 3:
			n0 := geom.Point{p[1], p[2]}
			n1 := geom.Point{p[3], p[4]}
			end := geom.Point{p[5], p[6]}
			for _, b := range extremitiesCubic(start, n0, n1, end) {
				include(b)
			}
			start = end
			p = p[8:]
		default:
			panic(fmt.Sprintf("raster: unexpected path segment type: %f", p[0]))
		}
		include(start)
	}

	return r
}

func clamp(x geom.Pt) geom.Pt {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func extremitiesQuad(n0, n1, n2 geom.Point) (b [2]geom.Point) {
	quad := func(t geom.Pt) (b geom.Point) {
		b.X = (1-t)*((1-t)*n0.X+t*n1.X) + t*((1-t)*n1.X+t*n2.X)
		b.Y = (1-t)*((1-t)*n0.Y+t*n1.Y) + t*((1-t)*n1.Y+t*n2.Y)
		return b
	}

	// A quadratic Bezier curve is defined over t ∈ [0, 1] as
	//
	//	B(t) = (1-t)*((1-t)*n0 + t*n1) + t*((1-t)*n1 + t*n2)
	//
	// Extremities are at values of t for B'(t)=0 and B''(t)=0.
	// The first derivative is
	//
	//	B'(t) = 2*(1-t)*(n1-n0) + 2*t*(n2-n1)
	//      0 = 2*(n1-n0)-2*(n1-n0)*t + 2*(n2-n1)*t
	//	-2*(n1-n0) = 2*t*(-n1+n0+n2-n1)
	//	n0-n1 / (n0+n2-2*n1) = t
	//
	// At B'(t) = 0 we get
	//
	//	t = (n0-n1) / (n0+n2-2*n1).
	//
	// The second is not a function of t, so we ignore it.
	t0 := clamp((n0.X - n1.X) / (n0.X + n2.X - 2*n1.X))
	t1 := clamp((n0.Y - n1.Y) / (n0.Y + n2.Y - 2*n1.Y))
	b[0] = quad(t0)
	b[1] = quad(t1)
	return b
}

func extremitiesCubic(n0, n1, n2, n3 geom.Point) [6]geom.Point {
	// A cubic Bezier curve is defined over t ∈ [0, 1] as
	//
	//	B(t) = (1-t)^3*n0 + 3*(1-t)^2*t*n1 + 3*(1-t)*t^2*n2 + t^3*p3
	//
	// Extremities are at values of t for B'(t)=0 and B''(t)=0.
	// The derivatives are
	//
	//	B'(t) = 3*(1-t)^2*(n1-n0) + 6*(1-t)*t*(n2-n1) + 3*t^2*(n3-n2)
	//	B''(t) = 6*(1-t)*(n2-2*n1+n0) + 6*t*(n3 - 2*n2 + n1)
	panic("bounds for cubics are not implemented")
}
