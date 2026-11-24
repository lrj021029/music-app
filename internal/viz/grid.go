package viz

import "math"

// The densities of the bars: how many there are, and how far apart.
const (
	DensityCompact  = "compact"
	DensityBalanced = "balanced"
	DensitySpacious = "spacious"
)

// MaxSegs is the most segments a bar has.
const MaxSegs = 48

// MinGap and MaxGap are how narrow and how wide a Layout's gaps may be.
const (
	MinGap = 0.05
	MaxGap = 0.7
)

// A Layout is how a spectrum's bars stand in their box: how many there
// are, the space beside each and above each of its segments, as a share of
// the room each has, and how tall a segment is with its gap.
type Layout struct {
	Bars   int
	BarGap float32
	SegGap float32
	Seg    float32 // in DIPs
}

// LayoutOf returns the layout of a density: Balanced's for one it doesn't
// know.
func LayoutOf(density string) Layout {
	switch density {
	case DensityCompact:
		return Layout{Bars: 40, BarGap: 0.22, SegGap: 0.24, Seg: 7}
	case DensitySpacious:
		return Layout{Bars: 22, BarGap: 0.48, SegGap: 0.36, Seg: 10}
	}
	return Layout{Bars: 32, BarGap: 0.36, SegGap: 0.3, Seg: 9}
}

// ClampGap returns a gap within MinGap and MaxGap.
func ClampGap(g float64) float64 { return min(max(g, MinGap), MaxGap) }

// A Grid is where a layout's bars, and their segments, go in a box: on
// whole pixels of the display, each a pixel or more from the next, and all
// inside the box, however small it is and however coarse the display. A
// box too narrow for the layout's bars has fewer, each showing the loudest
// of those it stands for.
type Grid struct {
	Bars, Segs  int
	x, y, scale float32 // the box's corner, in pixels
	pitch, gap  float32
	h, sp, sgap float32
}

// NewGrid lays out l in the box at x, y of w by h DIPs, on a display of
// scale pixels a DIP, and reports whether the box has room for any bars.
func NewGrid(x, y, w, h, scale float32, l Layout) (Grid, bool) {
	if scale <= 0 {
		scale = 1
	}
	x0, y0 := ceil(x*scale), ceil(y*scale)
	W, H := floor((x+w)*scale)-x0, floor((y+h)*scale)-y0
	// A bar takes 2 pixels at least, and its gap 1.
	n := min(l.Bars, MaxBars, int(W/3))
	segs := min(MaxSegs, max(8, int(H/max(l.Seg*scale, 3))), int(H/3))
	if n < 4 || segs < 8 {
		return Grid{}, false
	}
	pitch := W / float32(n)
	gap := min(max(round(pitch*float32(ClampGap(float64(l.BarGap)))), 1), floor(pitch-2))
	sp := H / float32(segs)
	sgap := min(max(round(sp*float32(ClampGap(float64(l.SegGap)))), 1), floor(sp-2))
	return Grid{Bars: n, Segs: segs, x: x0, y: y0, scale: scale, pitch: pitch, gap: gap, h: H, sp: sp, sgap: sgap}, true
}

// Bar returns where bar i is across, from the left: its left edge and its
// width, in DIPs.
func (g *Grid) Bar(i int) (x, w float32) {
	// The ends of two bars are a whole gap apart, as they round alike.
	a := round(float32(i)*g.pitch + g.gap/2)
	b := round(float32(i+1)*g.pitch - g.gap/2)
	return (g.x + a) / g.scale, (b - a) / g.scale
}

// Seg returns where segment i of a bar is, from its foot: its top and its
// height, in DIPs.
func (g *Grid) Seg(i int) (y, h float32) {
	foot := g.h - round(float32(i)*g.sp+g.sgap/2)
	top := g.h - round(float32(i+1)*g.sp-g.sgap/2)
	return (g.y + top) / g.scale, (foot - top) / g.scale
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }
func floor(v float32) float32 { return float32(math.Floor(float64(v))) }
func ceil(v float32) float32  { return float32(math.Ceil(float64(v) - 1e-4)) }
