// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sprite provides a 2D scene graph for rendering and animation.
//
// A tree of nodes is drawn by a rendering Engine, provided by another
// package. The OS-independent Go version based on the image package is:
//
//	code.google.com/p/go.mobile/sprite/portable
//
// An Engine draws a screen starting at a root Node. The tree is walked
// depth-first, with affine transformations applied at each level.
//
// Nodes are rendered relative to their parent.
//
// Typical main loop:
//
//	for each frame {
//		quantize time.Now() to a clock.Time
//		process UI events
//		modify the scene's nodes and animations (Arranger values)
//		e.Render(scene, t)
//	}
package sprite

import (
	"image"
	"image/draw"

	"github.com/crawshaw/sprite/clock"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
)

type Arranger interface {
	Arrange(e Engine, n *Node, t clock.Time)
}

type Texture interface {
	Bounds() (w, h int)
	Download(r image.Rectangle, dst draw.Image)
	Upload(r image.Rectangle, src image.Image)
	Unload()
}

type SubTex struct {
	T Texture
	R image.Rectangle
}

type Curve int32

type Engine interface {
	// LoadTexture loads a texture into the active Engine.
	LoadTexture(a image.Image) (Texture, error)

	// LoadPath loads a vector path into the active Engine.
	//
	// The path slice is an encoded sequence of tagged bezier
	// curve control points. Control points are geom.Pt co-ordinates.
	//  The first control point of a curve is the final control point
	// of the previous curve.
	//
	// Valid tags:
	//	{0, x, y}           - start control point
	//	{1, x, y}           - line segment control point
	//	{2, x1, y1, x2, y2} - quadratic segment control points
	//
	// TODO(crawshaw): support cubic segments
	// TODO(crawshaw): make []float32?
	LoadCurve(path []geom.Pt) (Curve, error)

	UnloadCurve(c Curve)

	Render(scene *Node, t clock.Time)
}

// TODO: 64-bit sizeof(Node): 8*7 + 8*2 + 8*2 + 4*4 = 104 bytes.
// is that a lot? a 100k empty node scene consumes 10MB. that's pushing it.

// A Node is a renderable element and forms a tree of Nodes.
type Node struct {
	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Node

	// Transform is an affine transformation matrix for this
	// node and its children.
	Transform *f32.Affine

	Arranger Arranger
	SubTex   SubTex
	Curve    Curve
}

// AppendChild adds a node c as a child of n.
//
// It will panic if c already has a parent or siblings.
func (n *Node) AppendChild(c *Node) {
	if c.Parent != nil || c.PrevSibling != nil || c.NextSibling != nil {
		panic("sprite: AppendChild called for an attached child Node")
	}
	last := n.LastChild
	if last != nil {
		last.NextSibling = c
	} else {
		n.FirstChild = c
	}
	n.LastChild = c
	c.Parent = n
	c.PrevSibling = last
}

// RemoveChild removes a node c that is a child of n. Afterwards, c will have
// no parent and no siblings.
//
// It will panic if c's parent is not n.
func (n *Node) RemoveChild(c *Node) {
	if c.Parent != n {
		panic("sprite: RemoveChild called for a non-child Node")
	}
	if n.FirstChild == c {
		n.FirstChild = c.NextSibling
	}
	if c.NextSibling != nil {
		c.NextSibling.PrevSibling = c.PrevSibling
	}
	if n.LastChild == c {
		n.LastChild = c.PrevSibling
	}
	if c.PrevSibling != nil {
		c.PrevSibling.NextSibling = c.NextSibling
	}
	c.Parent = nil
	c.PrevSibling = nil
	c.NextSibling = nil
}
