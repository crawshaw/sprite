// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"
	"log"
	"time"

	"golang.org/x/junk/msprite/clock"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/app/debug"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"

	"github.com/crawshaw/sprite"
	"github.com/crawshaw/sprite/glsprite"
	"github.com/crawshaw/sprite/raster"
)

var (
	start     = time.Now()
	lastClock = clock.Time(-1)

	eng   = glsprite.Engine()
	scene *sprite.Node

	curve      *quadraticBezier
	c0, c1, c2 *raster.Circle
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
	if now == lastClock {
		// TODO: figure out how to limit draw callbacks to 60Hz instead of
		// burning the CPU as fast as possible.
		// TODO: (relatedly??) sync to vblank?
		return
	}
	if last := time.Duration(now-lastClock) * time.Second / 60; last > 20*time.Millisecond {
		log.Printf("last = %v", last)
	}
	lastClock = now

	curve.n0 = c0.Center
	curve.n1 = c1.Center
	curve.n2 = c2.Center

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
		switch {
		case c0.Contains(t.Loc):
			selected = c0
		case c1.Contains(t.Loc):
			selected = c1
		case c2.Contains(t.Loc):
			selected = c2
		}
	case event.TouchMove:
		if selected != nil {
			selected.Center = t.Loc
		}
	case event.TouchEnd:
		selected = nil
	}
}

func loadScene() {
	scene = &sprite.Node{}
	addCircle := func(c *raster.Circle) {
		n := &sprite.Node{
			Drawable: &raster.Drawable{
				Shape: &raster.Stroke{
					Shape: c,
					Width: .5,
				},
				Color: color.RGBA{R: 0xff, A: 0xff},
			},
		}
		scene.AppendChild(n)
	}

	c0 = &raster.Circle{
		Center: geom.Point{10, 20},
		Radius: 3,
	}
	addCircle(c0)
	c1 = &raster.Circle{
		Center: geom.Point{30, 10},
		Radius: 3,
	}
	addCircle(c1)
	c2 = &raster.Circle{
		Center: geom.Point{50, 20},
		Radius: 3,
	}
	addCircle(c2)

	curve = new(quadraticBezier)
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
}

type quadraticBezier struct {
	n0, n1, n2 geom.Point
}

func (q quadraticBezier) Path() (p raster.Path) {
	p.AddStart(q.n0)
	p.AddQuadratic(q.n1, q.n2)
	return p
}
