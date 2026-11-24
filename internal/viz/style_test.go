package viz

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// An unlit segment, shown however plainly, is far dimmer than a lit one, in
// every theme and color: the bars never blur into the matrix behind them.
func TestUnlitSegmentsStayDim(t *testing.T) {
	light := func(c ui.Color) float32 {
		return float32(c.A) / 255 * (0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)) / 255
	}
	styles := map[string]Style{"album": AlbumStyle([]ui.Color{ui.RGB(220, 30, 40), ui.RGB(30, 60, 200)}), "album gray": AlbumStyle(nil)}
	for _, id := range []string{ThemeClassic, ThemeMinimal, ThemeApple, ThemeNeon} {
		styles[id] = StyleOf(id)
	}
	for c := range appleColors {
		styles["apple "+c] = AppleStyle(c)
	}
	for name, s := range styles {
		for _, mode := range []string{InactiveOff, InactiveSubtle, InactiveVisible} {
			for _, at := range []float32{0, 0.5, 1} {
				if off, lit := light(s.OffAt(at, mode)), light(s.At(at)); off > lit*0.3 {
					t.Errorf("%s, %s, at %v: unlit %v, lit %v", name, mode, at, off, lit)
				}
			}
		}
	}
}
