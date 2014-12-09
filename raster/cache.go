package raster

import (
	"fmt"
	"image"
	"image/color"

	"github.com/crawshaw/sprite"
	"github.com/crawshaw/sprite/clock"
)

// TODO: must make this adaptive
const colWidth = 128

type cacheEntry struct {
	id   sprite.Curve
	path Path
	b    image.Rectangle
	time clock.Time // needed for rendering at time

	next, prev *cacheEntry // linked-list, most recently used at front
}

type Cache struct {
	M     *image.RGBA // TODO: *image.Alpha??
	Dirty bool

	cache      map[sprite.Curve]*cacheEntry
	cacheFront *cacheEntry // front of cacheEntry linked-list
	x, y       int         // next empty slot
}

func (c *Cache) Get(id sprite.Curve, p Path, t clock.Time) (image.Rectangle, error) {
	if c.cache == nil {
		c.cache = make(map[sprite.Curve]*cacheEntry)
	}
	entry, err := c.get(id, p, t)
	return entry.b, err
}

func (c *Cache) get(id sprite.Curve, p Path, t clock.Time) (*cacheEntry, error) {
	entry := c.cache[id]
	if entry == nil {
		entry = &cacheEntry{id: id, path: p}
		if err := c.rasterize(entry, t); err != nil {
			return nil, err
		}
		c.cache[id] = entry
		c.Dirty = true
	} else {
		entry.time = t

		// remove from list
		if entry.prev != nil {
			entry.prev.next = entry.next
		}
		if entry.next != nil {
			entry.next.prev = entry.prev
		}
	}

	// put on front of list
	entry.prev = nil
	entry.next = c.cacheFront
	if c.cacheFront != nil {
		c.cacheFront.prev = entry
		c.cacheFront = entry
	}
	return entry, nil
}

func (c *Cache) findSpace(w, h int, t clock.Time) (image.Point, error) {
	if w > colWidth {
		return image.Point{}, fmt.Errorf("raster: curve larger than cache column width: %d", w)
	}
	b := c.M.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if h > sh-c.y {
		c.x += colWidth
		c.y = 0
	}
	if c.x >= sw {
		// out of space, clear out old curves
		if err := c.clearHalf(t); err != nil {
			return image.Point{}, err
		}
		if h > sh-c.y {
			c.x += colWidth
			c.y = 0
		}
	}
	if w > sw-c.x || h > sh-c.y {
		return image.Point{}, fmt.Errorf("raster: no space for curve w=%d, h=%d", w, h)
	}
	p := image.Point{c.x, c.y}
	c.y += h
	return p, nil
}

func (c *Cache) clearHalf(t clock.Time) error {
	e := c.cacheFront
	for e.next != nil {
		e = e.next
	}

	toDelete := len(c.cache) / 2
	deleted := 0
	for e != nil && toDelete > 0 {
		if e.time < t {
			delete(c.cache, e.id)
			if e.next != nil {
				e.next.prev = e.prev
			}
			if e.prev != nil {
				e.prev.next = e.next
			}
			deleted++
			toDelete--
		}
		e = e.prev
	}
	if deleted == 0 {
		return fmt.Errorf("raster: curve cache is full (%d items)", len(c.cache))
	}

	// re-render cache
	for e := c.cacheFront; e != nil; e = e.next {
		if err := c.rasterize(e, e.time); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) rasterize(entry *cacheEntry, t clock.Time) error {
	b := entry.path.Bounds()
	w := int((b.Max.X - b.Min.X).Px() + 0.5)
	h := int((b.Max.Y - b.Min.Y).Px() + 0.5)
	p, err := c.findSpace(w, h, t)
	if err != nil {
	}
	entry.b = image.Rect(p.X, p.Y, p.X+w, p.Y+h)
	m := c.M.SubImage(entry.b).(*image.RGBA)
	for i := range m.Pix {
		m.Pix[i] = 0
	}
	Draw(m, &Drawable{&entry.path, color.Black})
	/*
		entry.texture = sprite.SubTex{
			T: c.s.s,
			R: image.Rect(p.X, p.Y, p.X+w, p.Y+h),
		}
		c.s.s.Upload(entry.texture.R, a)
	*/
	return nil
}
