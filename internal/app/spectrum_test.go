package app

import (
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
