// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package glsprite blah blah blah TODO.
package glsprite

import (
	"image"
	"image/draw"
	"log"

	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl/glutil"

	"github.com/crawshaw/sprite"
	"github.com/crawshaw/sprite/clock"
	"github.com/crawshaw/sprite/raster"
)

type texture struct {
	glImage *glutil.Image
	b       image.Rectangle
}

func (t *texture) Bounds() (w, h int) { return t.b.Dx(), t.b.Dy() }

func (t *texture) Download(r image.Rectangle, dst draw.Image) {
	panic("TODO")
}

func (t *texture) Upload(r image.Rectangle, src image.Image) {
	draw.Draw(t.glImage.RGBA, r, src, src.Bounds().Min, draw.Src)
	t.glImage.Upload()
}

func (t *texture) Unload() {
	panic("TODO")
}

func Engine() sprite.Engine {
	return &engine{
		curves:      make(map[sprite.Curve]raster.Path),
		curveBounds: make(map[sprite.Curve]*f32.Affine),
		nextCurve:   1,
	}
}

type engine struct {
	raster        *glutil.Image
	rasterCache   *raster.Cache
	absTransforms []f32.Affine

	curves      map[sprite.Curve]raster.Path
	curveBounds map[sprite.Curve]*f32.Affine
	nextCurve   int32
}

func (e *engine) LoadTexture(src image.Image) (sprite.Texture, error) {
	b := src.Bounds()
	t := &texture{glutil.NewImage(b.Dx(), b.Dy()), b}
	t.Upload(b, src)
	// TODO: set "glImage.Pix = nil"?? We don't need the CPU-side image any more.
	return t, nil
}

func (e *engine) LoadCurve(path []geom.Pt) (sprite.Curve, error) {
	id := sprite.Curve(e.nextCurve)
	e.nextCurve++
	e.curves[id] = raster.Path(path)

	if e.raster == nil {
		// TODO: round up to power of two.
		// TODO: screen size is a proxy for sensible amount of memory
		// to spend. determine a better bound.
		w := int(geom.Width.Px()+0.5) * 2
		h := int(geom.Height.Px()+0.5) * 2
		e.raster = glutil.NewImage(w, h)
		e.rasterCache = &raster.Cache{M: e.raster.RGBA}
	}
	// TODO there is no need to render curves now, it can be done lazily
	// so we can have far more loaded than we have seaparate texture space
	// for. But this gives us a nice way to propagate errors until the
	// cache is properly built.
	log.Printf("LoadCurve: path=%v", path)
	_, err := e.rasterCache.Get(id, path, 0)
	if err != nil {
		return 0, err
	}

	b := raster.Path(path).Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y
	e.curveBounds[id] = &f32.Affine{
		{float32(w), 0, float32(b.Min.X)},
		{0, float32(h), float32(b.Min.Y)},
	}
	return id, nil
}

func (e *engine) UnloadCurve(c sprite.Curve) {
	delete(e.curves, c)
	delete(e.curveBounds, c)
}

func (e *engine) Render(scene *sprite.Node, t clock.Time) {
	e.absTransforms = append(e.absTransforms[:0], f32.Affine{
		{1, 0, 0},
		{0, 1, 0},
	})
	e.render(scene, t)
}

func (e *engine) render(n *sprite.Node, t clock.Time) {
	if n.Arranger != nil {
		n.Arranger.Arrange(e, n, t)
	}

	// Push absTransforms.
	m := e.absTransforms[len(e.absTransforms)-1]
	if n.Transform != nil {
		m.Mul(&m, n.Transform)
		e.absTransforms = append(e.absTransforms, m)
	}

	if x := n.SubTex; x.T != nil {
		x.T.(*texture).glImage.Draw(
			geom.Point{
				geom.Pt(m[0][2]),
				geom.Pt(m[1][2]),
			},
			geom.Point{
				geom.Pt(m[0][2] + m[0][0]),
				geom.Pt(m[1][2] + m[1][0]),
			},
			geom.Point{
				geom.Pt(m[0][2] + m[0][1]),
				geom.Pt(m[1][2] + m[1][1]),
			},
			x.R,
		)
	}

	if n.Curve != 0 {
		m.Mul(&m, e.curveBounds[n.Curve])

		b, err := e.rasterCache.Get(n.Curve, e.curves[n.Curve], 0)
		if err != nil {
			panic(err)
		}
		if e.rasterCache.Dirty {
			// given the current cache design this should never happen
			// TODO: when it does happen deliberately, delay e.raster.Draw
			// calls so they are executed in batches with a single Upload.
			log.Println("upload raster on draw")
			e.raster.Upload()
			e.rasterCache.Dirty = false
		}

		e.raster.Draw(
			geom.Point{
				geom.Pt(m[0][2]),
				geom.Pt(m[1][2]),
			},
			geom.Point{
				geom.Pt(m[0][2] + m[0][0]),
				geom.Pt(m[1][2] + m[1][0]),
			},
			geom.Point{
				geom.Pt(m[0][2] + m[0][1]),
				geom.Pt(m[1][2] + m[1][1]),
			},
			b,
		)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e.render(c, t)
	}

	if n.Transform != nil {
		e.absTransforms = e.absTransforms[:len(e.absTransforms)-1]
	}

}
