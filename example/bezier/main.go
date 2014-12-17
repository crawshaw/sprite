// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/app/debug"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"

	"github.com/crawshaw/sprite"
	"github.com/crawshaw/sprite/clock"
	"github.com/crawshaw/sprite/glsprite"
	"github.com/crawshaw/sprite/raster"
)

var (
	start     = time.Now()
	lastClock = clock.Time(-1)

	eng   = glsprite.Engine()
	scene *sprite.Node

	bounds     *raster.Rectangle
	curve      *quadraticBezier
	c0, c1, c2 *sprite.Node
	selected   *raster.Circle
)

func main() {
	app.Run(app.Callbacks{
		Draw:  draw,
		Touch: touch,
	})
}

func draw() {
	if scene == nil {
		loadScene()
	}

	now := clock.Time(time.Since(start) * 60 / time.Second)
	lastClock = now

	/*
		curve.n0 = c0.Center
		curve.n1 = c1.Center
		curve.n2 = c2.Center
		*bounds = raster.Rectangle(curve.Path().Bounds())
		log.Printf("curve: %v, bounds: %v", curve, *bounds)
		log.Printf("geom W=%v, H=%v", geom.Width, geom.Height)
	*/

	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)
	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	eng.Render(scene, 0)
	debug.DrawFPS()
}

func touch(t event.Touch) {
	switch t.Type {
	case event.TouchStart:
		/*
			switch {
			case c0.Contains(t.Loc):
				selected = c0
			case c1.Contains(t.Loc):
				selected = c1
			case c2.Contains(t.Loc):
				selected = c2
			}
		*/
	case event.TouchMove:
		if selected != nil {
			// TODO selected.Center = t.Loc
		}
	case event.TouchEnd:
		selected = nil
	}
}

func loadScene() {
	scene = &sprite.Node{}

	circlePath := (&raster.Stroke{
		Shape: &raster.Circle{
			Radius: 3,
		},
		Width: 0.5,
	}).Path()
	// TODO put somewhere.
	b := circlePath.Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y
	circleAffine := f32.Affine{
		{float32(w), 0, float32(b.Min.X)},
		{0, float32(h), float32(b.Min.Y)},
	}
	circleCurve, err := eng.LoadCurve(circlePath)
	if err != nil {
		panic(err)
	}

	addCircle := func(x, y geom.Pt) *sprite.Node {
		t := circleAffine
		t[0][2] += float32(x)
		t[1][2] += float32(y)
		n := &sprite.Node{
			Transform: &t,
			Curve:     circleCurve,
			// TODO Color: color.RGBA{R: 0xff, A: 0xff},
		}
		scene.AppendChild(n)
		return n
	}

	c0 = addCircle(10, 20)
	c1 = addCircle(30, 10)
	c2 = addCircle(50, 20)

	curve = new(quadraticBezier)
	curveNode := &sprite.Node{
	// TODO: Color: color.Black
	}
	scene.AppendChild(curveNode)

	/*
		n := &sprite.Node{
			Drawable: &raster.Drawable{
				Shape: &raster.Stroke{
					Shape: curve,
					Width: .7,
				},
				Color: color.Black,
			},
		}
		scene.AppendChild(n)
	*/

	bounds = new(raster.Rectangle)
	//boundsNode := &sprite.Node{
	// TODO Color: color.Gray{0xdd},
	//}
	/*
		n := &sprite.Node{
			Drawable: &raster.Drawable{
				Shape: &raster.Stroke{
					Shape: bounds,
					Width: .5,
				},
			},
		}
		scene.AppendChild(n)
	*/
}

type quadraticBezier struct {
	n0, n1, n2 geom.Point
}

func (q quadraticBezier) Path() (p raster.Path) {
	p.AddStart(q.n0)
	p.AddQuadratic(q.n1, q.n2)
	return p
}
