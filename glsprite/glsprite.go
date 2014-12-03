// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package glsprite blah blah blah TODO.
package glsprite

import (
	"image"
	"image/draw"

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
	return &engine{}
}

type engine struct {
	raster        *glutil.Image
	absTransforms []f32.Affine
}

func (e *engine) LoadTexture(src image.Image) (sprite.Texture, error) {
	b := src.Bounds()
	t := &texture{glutil.NewImage(b.Dx(), b.Dy()), b}
	t.Upload(b, src)
	// TODO: set "glImage.Pix = nil"?? We don't need the CPU-side image any more.
	return t, nil
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

	if n.Drawable != nil {
		if n.Transform == nil {
			// Give ourselves the space of the screen for drawing.
			// TODO: use smaller bounding box.
			m.Mul(&m, &f32.Affine{
				{float32(geom.Width), 0, 0},
				{0, float32(geom.Height), 0},
			})
		}
		if e.raster == nil {
			w := int(geom.Width.Px() + 0.5)
			h := int(geom.Height.Px() + 0.5)
			e.raster = glutil.NewImage(w, h)
		}
		w := int(geom.Pt(m[0][0]).Px() + 0.5)
		h := int(geom.Pt(m[1][1]).Px() + 0.5)
		b := image.Rect(0, 0, w, h)
		scratch := e.raster.RGBA.SubImage(b).(*image.RGBA)
		clear := scratch.Pix[0:scratch.PixOffset(w, h)]
		for i := range clear {
			clear[i] = 0
		}
		raster.Draw(scratch, n.Drawable)
		e.raster.Upload()
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
