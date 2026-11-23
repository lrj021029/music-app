package viz

import (
	"cmp"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// The themes of the spectrum, by their ids. Classic LED is the default;
// Minimal is of its family. Album Adaptive takes a cover's colors
// (AlbumStyle).
const (
	ThemeClassic = "classic"
	ThemeMinimal = "minimal"
	ThemeApple   = "apple"
	ThemeNeon    = "neon"
	ThemeAlbum   = "album"
)

// A Stop is a color of a style's bars, at a height from 0, their foot, to
// 1, their top.
type Stop struct {
	At    float32
	Color ui.Color
}

// A Style is how a spectrum's bars are colored: a color at each height,
// the segments not lit, the peaks, and a glow around what is lit.
type Style struct {
	stops [6]Stop
	n     int
	Off   ui.Color // a segment not lit, none if clear
	Peak  ui.Color // a peak's; clear, the color where it is, lighter
	Glow  float32  // how strong the glow around the lit segments is, 0 for none
}

var white = ui.RGB(255, 255, 255)

func style(off, peak ui.Color, glow float32, stops ...Stop) Style {
	s := Style{Off: off, Peak: peak, Glow: glow}
	s.n = copy(s.stops[:], stops)
	return s
}

// At returns the color of a lit segment at height t.
func (s *Style) At(t float32) ui.Color {
	if s.n == 0 {
		return white
	}
	t = min(max(t, 0), 1)
	for i := 1; i < s.n; i++ {
		if a, b := s.stops[i-1], s.stops[i]; t <= b.At {
			if b.At <= a.At || t <= a.At {
				return b.Color
			}
			return a.Color.Mix(b.Color, (t-a.At)/(b.At-a.At))
		}
	}
	return s.stops[s.n-1].Color
}

// PeakAt returns the color of a peak at height t.
func (s *Style) PeakAt(t float32) ui.Color {
	if s.Peak.A != 0 {
		return s.Peak
	}
	return s.At(t).Mix(white, 0.4)
}

// StyleOf returns the style of a theme: Classic LED's for one it doesn't
// know, and for Album Adaptive, which needs a cover.
func StyleOf(id string) Style {
	switch id {
	case ThemeMinimal:
		c := ui.RGB(206, 211, 220)
		return style(c.Alpha(0.05), white.Alpha(0.9), 0, Stop{0, c.Alpha(0.5)}, Stop{1, c.Alpha(0.92)})
	case ThemeApple:
		return style(white.Alpha(0.05), ui.Color{}, 0.25,
			Stop{0, ui.Hex("#009DDC")}, Stop{0.2, ui.Hex("#963D97")}, Stop{0.4, ui.Hex("#E03A3E")},
			Stop{0.6, ui.Hex("#F5821F")}, Stop{0.8, ui.Hex("#FDB827")}, Stop{1, ui.Hex("#61BB46")})
	case ThemeNeon:
		return style(ui.RGBA(120, 90, 255, 0.07), ui.Hex("#F4F0FF"), 0.5,
			Stop{0, ui.Hex("#00E5FF")}, Stop{0.4, ui.Hex("#3D7BFF")}, Stop{0.75, ui.Hex("#8A4DFF")}, Stop{1, ui.Hex("#FF3DF2")})
	}
	return style(white.Alpha(0.06), ui.Color{}, 0.3,
		Stop{0, ui.Hex("#1ED760")}, Stop{0.3, ui.Hex("#2ED3C6")}, Stop{0.55, ui.Hex("#F5E14B")},
		Stop{0.78, ui.Hex("#FF9F2E")}, Stop{1, ui.Hex("#FF3B30")})
}

// AlbumStyle returns the style of a cover's colors, as Palette found them,
// the most of it first: up to three of its vivid colors, the darker at the
// foot of the bars, each lit enough to show over the dark. A cover without
// color has its own gray, rising to white.
func AlbumStyle(pal []ui.Color) Style {
	type swatch struct{ h, s, v float64 }
	var picks []swatch
	for _, c := range pal {
		h, s, v := hsv(c)
		if s < 0.22 || v < 0.12 {
			continue
		}
		if slices.ContainsFunc(picks, func(p swatch) bool { return hueDist(p.h, h) < 28 }) {
			continue
		}
		if picks = append(picks, swatch{h, s, v}); len(picks) == 3 {
			break
		}
	}
	if len(picks) == 0 {
		base := ui.RGB(150, 153, 162)
		if len(pal) > 0 {
			h, s, v := hsv(pal[0])
			base = rgb(h, min(s, 0.25), max(v, 0.55))
		}
		return style(base.Alpha(0.07), white.Alpha(0.95), 0.2, Stop{0, base}, Stop{1, base.Mix(white, 0.7)})
	}
	slices.SortFunc(picks, func(a, b swatch) int { return cmp.Compare(a.v, b.v) })
	stops := make([]Stop, 0, 4)
	for i, p := range picks {
		at := float32(0)
		if len(picks) > 1 {
			at = float32(i) / float32(len(picks)-1) * 0.85
		}
		v := min(max(p.v, 0.62+0.14*float64(i)), 1)
		stops = append(stops, Stop{at, rgb(p.h, min(max(p.s, 0.45), 0.85), v)})
	}
	top := stops[len(stops)-1].Color
	stops = append(stops, Stop{1, top.Mix(white, 0.3)})
	return style(stops[0].Color.Alpha(0.08), ui.Color{}, 0.35, stops...)
}

func hueDist(a, b float64) float64 {
	d := math.Abs(a - b)
	return min(d, 360-d)
}

func hsv(c ui.Color) (h, s, v float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := max(r, g, b), min(r, g, b)
	if v = mx; mx == 0 {
		return 0, 0, 0
	}
	d := mx - mn
	s = d / mx
	switch {
	case d == 0:
	case mx == r:
		h = math.Mod((g-b)/d, 6) * 60
	case mx == g:
		h = ((b-r)/d + 2) * 60
	default:
		h = ((r-g)/d + 4) * 60
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}

func rgb(h, s, v float64) ui.Color {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g = c, x
	case h < 120:
		r, g = x, c
	case h < 180:
		g, b = c, x
	case h < 240:
		g, b = x, c
	case h < 300:
		r, b = x, c
	default:
		r, b = c, x
	}
	to := func(f float64) uint8 { return uint8(math.Round(min(max(f+m, 0), 1) * 255)) }
	return ui.RGB(to(r), to(g), to(b))
}
