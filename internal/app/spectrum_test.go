package app

import (
	"math"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
	"mygo-music/internal/viz"
)

// A click on a theme's chip under the spectrum colors its bars so, and the
// settings keep it.
func TestSpectrumChipChoosesTheme(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "宝船")})
	a.pl.SetLyrics(nil)
	a.nowPlaying, a.npPane = true, paneSpectrum
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	if err := tt.Click("Album Adaptive"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.settings.SpectrumTheme != viz.ThemeAlbum {
		t.Errorf("theme %q after the click, want %q", a.settings.SpectrumTheme, viz.ThemeAlbum)
	}
}

// A click on a density's chip spaces the bars so, and a theme chosen after
// keeps it, rather than putting back its own.
func TestSpectrumDensityChipOutlastsTheme(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "宝船")})
	a.pl.SetLyrics(nil)
	a.nowPlaying, a.npPane = true, paneSpectrum
	tt := ui.NewTester(a.view, 1400, 800)
	settled(tt)
	if err := tt.Click("Spacious"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Neon"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if d := a.spectrumDensity(); d != viz.DensitySpacious {
		t.Errorf("density %q after choosing Spacious, then Neon", d)
	}
	if l := a.spectrumLayout(); l.Bars != viz.LayoutOf(viz.DensitySpacious).Bars {
		t.Errorf("%d bars, Spacious", l.Bars)
	}
}

// Gaps of the listener's own make the density Custom and hold, whatever
// theme or other gap is chosen after; choosing a preset puts its gaps back.
func TestSpectrumCustomGapsHold(t *testing.T) {
	a := newTestApp()
	a.settings.SpectrumTheme = viz.ThemeMinimal // Spacious of its own
	a.setSpectrumGap(true, 0.6)
	a.setSpectrumGap(false, 0.12)
	a.settings.SpectrumTheme = viz.ThemeClassic
	l := a.spectrumLayout()
	if a.spectrumDensity() != densityCustom || math.Abs(float64(l.BarGap)-0.6) > 1e-6 || math.Abs(float64(l.SegGap)-0.12) > 1e-6 {
		t.Errorf("after the gaps and another theme: %q, %+v", a.spectrumDensity(), l)
	}
	if l.Bars != viz.LayoutOf(viz.DensitySpacious).Bars {
		t.Errorf("%d bars, not the Spacious ones the gaps were set over", l.Bars)
	}
	a.setSpectrumDensity(viz.DensityCompact)
	if l := a.spectrumLayout(); l != viz.LayoutOf(viz.DensityCompact) {
		t.Errorf("Compact chosen after: %+v", l)
	}
}

// With MM_FRAMES set, saves the spectrum at each density, unlit segments
// each way and in each theme, of a song's spectrum held still.
func TestSpectrumFrames(t *testing.T) {
	dir := os.Getenv("MM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	f := &viz.Frame{Bars: viz.MaxBars}
	for b := range f.Bars {
		x := float64(b) / float64(f.Bars-1)
		v := 0.75*math.Exp(-math.Pow((x-0.3)/0.22, 2)) + 0.3*math.Abs(math.Sin(x*17)) - 0.12*x
		f.Levels[b] = float32(min(max(v, 0), 1))
		f.Peaks[b] = min(f.Levels[b]+0.08, 1)
	}
	shots := []struct{ name, theme, density, inactive string }{
		{"compact", viz.ThemeClassic, viz.DensityCompact, ""},
		{"balanced", viz.ThemeClassic, viz.DensityBalanced, ""},
		{"spacious", viz.ThemeClassic, viz.DensitySpacious, ""},
		{"balanced-visible", viz.ThemeClassic, viz.DensityBalanced, viz.InactiveVisible},
		{"balanced-off", viz.ThemeApple, viz.DensityBalanced, viz.InactiveOff},
		{"spacious-neon", viz.ThemeNeon, viz.DensitySpacious, ""},
		{"minimal", viz.ThemeMinimal, "", ""},
	}
	for _, s := range shots {
		a := newTestApp()
		a.pl.Cue([]music.Track{song("1", "宝船")})
		a.pl.SetLyrics(nil)
		a.nowPlaying, a.npPane = true, paneSpectrum
		a.settings.SpectrumTheme, a.settings.SpectrumDensity, a.settings.SpectrumInactive = s.theme, s.density, s.inactive
		a.spectrum.shown = f
		tt := ui.NewTester(a.view, 1200, 760)
		tt.SetScale(1.5)
		settled(tt)
		tt.Advance(spectrumFade)
		tt.Frame()
		saveFrame(t, dir, "spectrum-"+s.name+".png", tt.Image())
	}
}

// Turned off in the settings, the spectrum's tab goes, and Now Playing shows
// the lyrics in its place rather than an empty pane.
func TestSpectrumOffLeavesItsPane(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "宝船")})
	a.pl.SetLyrics(nil)
	a.nowPlaying, a.npPane = true, paneSpectrum
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	a.settings.Spectrum = false
	tt.Frame()
	if a.npPane == paneSpectrum {
		t.Error("the spectrum still shows, turned off")
	}
	for _, tab := range a.npTabs() {
		if tab.pane == paneSpectrum {
			t.Error("the spectrum's tab still shows, turned off")
		}
	}
}
