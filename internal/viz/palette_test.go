package viz

import (
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

func near(a, b ui.Color) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R) < 24 && d(a.G, b.G) < 24 && d(a.B, b.B) < 24
}

// A cover of two colors gives both, the one covering more of it first.
func TestPaletteFindsACoversColors(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	red, blue := color.RGBA{220, 30, 40, 255}, color.RGBA{30, 60, 200, 255}
	for y := range 300 {
		for x := range 300 {
			c := red
			if x >= 200 {
				c = blue
			}
			img.SetRGBA(x, y, c)
		}
	}
	pal := Palette(img, 5)
	if len(pal) < 2 || !near(pal[0], ui.RGB(220, 30, 40)) || !near(pal[1], ui.RGB(30, 60, 200)) {
		t.Fatalf("palette %v, want red, then blue", pal)
	}
}

// A gray or a dark cover still lights its bars over the dark: Album
// Adaptive never makes them too dim to see.
func TestAlbumStyleShowsOverTheDark(t *testing.T) {
	for _, pal := range [][]ui.Color{nil, {ui.RGB(20, 20, 22)}, {ui.RGB(40, 8, 10), ui.RGB(10, 10, 60)}} {
		s := AlbumStyle(pal)
		if _, _, v := hsv(s.At(0)); v < 0.5 {
			t.Errorf("%v: the foot of the bars is %v", pal, s.At(0))
		}
	}
}
