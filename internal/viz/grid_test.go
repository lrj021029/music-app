package viz

import (
	"math"
	"testing"
)

// However big the box and however fine the display, the bars and their
// segments stay inside it, each a pixel or more from the next, at every
// density and at the widest and narrowest gaps.
func TestGridKeepsBarsApartAndInside(t *testing.T) {
	layouts := []Layout{LayoutOf(DensityCompact), LayoutOf(DensityBalanced), LayoutOf(DensitySpacious),
		{Bars: MaxBars, BarGap: MaxGap, SegGap: MaxGap, Seg: 4}, {Bars: MaxBars, BarGap: MinGap, SegGap: MinGap, Seg: 4}}
	px := func(v, s float32) float64 { return math.Round(float64(v * s)) }
	for _, l := range layouts {
		for _, s := range []float32{1, 1.25, 1.5, 1.75, 2, 3} {
			for _, sz := range [][2]float32{{120, 90}, {333.3, 161}, {640, 420}, {1001.7, 777.2}, {2400, 1300}} {
				x, y, w, h := float32(17.3), float32(41.6), sz[0], sz[1]
				g, ok := NewGrid(x, y, w, h, s, l)
				if !ok {
					t.Errorf("%v at %vx%v, scale %v: no room", l, w, h, s)
					continue
				}
				if g.Bars > l.Bars {
					t.Errorf("%v: %d bars", l, g.Bars)
				}
				last := math.Floor(float64(x*s)) - 1
				for i := range g.Bars {
					bx, bw := g.Bar(i)
					l0, r0 := px(bx, s), px(bx+bw, s)
					if l0 < last+1 || r0-l0 < 1 {
						t.Fatalf("%v at %vx%v, scale %v: bar %d from %v to %v, the last ended at %v", l, w, h, s, i, l0, r0, last)
					}
					last = r0
				}
				if last > math.Floor(float64((x+w)*s)) {
					t.Errorf("%v at %vx%v, scale %v: the bars end at %v, past the box", l, w, h, s, last)
				}
				above := math.Ceil(float64((y+h)*s)) + 1
				for i := range g.Segs {
					sy, sh := g.Seg(i)
					top, foot := px(sy, s), px(sy+sh, s)
					if foot > above-1 || foot-top < 1 {
						t.Fatalf("%v at %vx%v, scale %v: segment %d from %v to %v, the last began at %v", l, w, h, s, i, top, foot, above)
					}
					above = top
				}
				if above < math.Floor(float64(y*s)) {
					t.Errorf("%v at %vx%v, scale %v: the segments reach %v, above the box", l, w, h, s, above)
				}
			}
		}
	}
}
