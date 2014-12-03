package raster

import (
	"testing"

	"golang.org/x/mobile/geom"
)

func TestBounds(t *testing.T) {
	var tests = []struct {
		curve [3]geom.Point
		want  geom.Rectangle
	}{
		{
			curve: [3]geom.Point{{10, 20}, {150, 40}, {200, 100}},
			want:  geom.Rectangle{geom.Point{10, 20}, geom.Point{200, 100}},
		},
		{
			curve: [3]geom.Point{{18, 102}, {166, 183}, {55, 115}},
			want:  geom.Rectangle{geom.Point{18, 102}, geom.Point{102.57, 146.03}},
		},
		{
			curve: [3]geom.Point{{200, 73}, {52, 186}, {220, 85}},
			want:  geom.Rectangle{geom.Point{130.68, 73}, geom.Point{220, 132.67}},
		},
		{
			curve: [3]geom.Point{{130, 36}, {200, 156}, {223, 69}},
			want:  geom.Rectangle{geom.Point{130, 36}, geom.Point{223, 105.57}},
		},
	}

	const epsilon = 0.01
	eq := func(x, y geom.Pt) bool {
		diff := x - y
		if diff < 0 {
			diff = -diff
		}
		return diff < epsilon
	}
	pointEq := func(x, y geom.Point) bool { return eq(x.X, y.X) && eq(x.Y, y.Y) }
	rectEq := func(x, y geom.Rectangle) bool { return pointEq(x.Min, y.Min) && pointEq(x.Max, y.Max) }

	for _, test := range tests {
		p := new(Path)
		p.AddStart(test.curve[0])
		p.AddQuadratic(test.curve[1], test.curve[2])
		got := p.Bounds()
		if !rectEq(got, test.want) {
			t.Errorf("%v: got bounds %v, want %v", test.curve, got, test.want)
		}
	}
}
