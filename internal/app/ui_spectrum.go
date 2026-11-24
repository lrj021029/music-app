package app

import (
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/viz"
)

// The spectrum is a pane of Now Playing: bars of LED segments, as a hi-fi's
// equalizer has, measured from what the player decodes on a goroutine of
// their own (viz.Runner) and drawn in one box. Nothing measures while
// nothing shows them: closed, hidden or turned off, the runner stops, and
// the player keeps nothing for it.

// spectrumThemes are the spectrum's themes, as the settings and the chips
// under it name them.
var spectrumThemes = []struct{ id, label string }{
	{viz.ThemeClassic, N("Classic LED")},
	{viz.ThemeMinimal, N("Minimal")},
	{viz.ThemeApple, N("Apple Colors")},
	{viz.ThemeNeon, N("Neon")},
	{viz.ThemeAlbum, N("Album Adaptive")},
}

// spectrumRates are how often the bars may move.
var spectrumRates = []struct {
	fps   int
	label string
}{
	{30, N("30 FPS")},
	{60, N("60 FPS")},
}

// densityCustom is the density of the listener's own gaps.
const densityCustom = "custom"

// spectrumDensities are how many bars there are and how far apart: the
// theme's own, three presets, and the listener's own gaps. The chips under
// the spectrum offer the presets.
var spectrumDensities = []struct{ id, label string }{
	{"", N("Theme Default")},
	{viz.DensityCompact, N("Compact")},
	{viz.DensityBalanced, N("Balanced")},
	{viz.DensitySpacious, N("Spacious")},
	{densityCustom, N("Custom")},
}

// spectrumInactives are how the segments not lit may show.
var spectrumInactives = []struct{ id, label string }{
	{"", N("Theme Default")},
	{viz.InactiveOff, N("Off")},
	{viz.InactiveSubtle, N("Subtle")},
	{viz.InactiveVisible, N("Visible")},
}

// spectrumAppleColors are the colors of Apple Colors (viz.AppleStyle).
var spectrumAppleColors = []struct{ id, label string }{
	{"", N("Multicolor")},
	{"blue", N("Blue")},
	{"purple", N("Purple")},
	{"pink", N("Pink")},
	{"green", N("Green")},
	{"orange", N("Orange")},
	{"cyan", N("Cyan")},
}

const (
	// spectrumFade is how long the bars take to go from one style's
	// colors to the next: another theme, or another cover's.
	spectrumFade = 600 * time.Millisecond
	// spectrumStillRate is how often they move with Reduce Motion on,
	// their peaks left out.
	spectrumStillRate = 12
)

// spectrumState is the spectrum as it was last drawn.
type spectrumState struct {
	run       *viz.Runner
	fps, bars int // the runner's
	frame     viz.Frame
	// The style the bars fade from, and to, since fadeAt.
	from, to viz.Style
	fadeAt   time.Time
	started  bool
	// The cover's style, for Album Adaptive, and the cover it is of.
	album    viz.Style
	albumArt *artwork
	albumSet bool
	// The segments' colors, lit and not, foot to top.
	cols, offs [viz.MaxSegs]ui.Color
	// The gaps the settings' sliders hold between their moves.
	barGap, segGap float64
	gapsShown      bool
	// shown, if set, is drawn rather than what plays, measured: a test's.
	shown *viz.Frame
}

func (a *app) spectrumTheme() string {
	for _, t := range spectrumThemes {
		if t.id == a.settings.SpectrumTheme {
			return t.id
		}
	}
	return viz.ThemeClassic
}

func (a *app) spectrumRate(still bool) int {
	switch {
	case still:
		return spectrumStillRate
	case a.settings.SpectrumFPS >= 60:
		return 60
	}
	return 30
}

// spectrumDensity returns the density of the bars: the listener's, or the
// theme's while they chose none.
func (a *app) spectrumDensity() string {
	switch d := a.settings.SpectrumDensity; d {
	case viz.DensityCompact, viz.DensityBalanced, viz.DensitySpacious:
		return d
	case densityCustom:
		if a.settings.SpectrumBars > 0 {
			return d
		}
	}
	return viz.ThemeDensity(a.spectrumTheme())
}

// spectrumLayout returns how the bars stand: a preset's layout, or the
// listener's gaps over the bars of the preset they started from.
func (a *app) spectrumLayout() viz.Layout {
	d := a.spectrumDensity()
	if d != densityCustom {
		return viz.LayoutOf(d)
	}
	s := &a.settings
	near := viz.DensityBalanced
	switch {
	case s.SpectrumBars >= 36:
		near = viz.DensityCompact
	case s.SpectrumBars <= 27:
		near = viz.DensitySpacious
	}
	l := viz.LayoutOf(near)
	l.Bars = min(max(s.SpectrumBars, 8), viz.MaxBars)
	l.BarGap, l.SegGap = float32(viz.ClampGap(s.SpectrumBarGap)), float32(viz.ClampGap(s.SpectrumSegGap))
	return l
}

// setSpectrumDensity chooses a density; Custom starts from the gaps shown.
func (a *app) setSpectrumDensity(id string) {
	if id == densityCustom {
		l := a.spectrumLayout()
		a.setSpectrumGap(true, float64(l.BarGap))
		return
	}
	a.settings.SpectrumDensity = id
}

// setSpectrumGap sets the gap beside the bars, or else above their
// segments, keeping the bars and the other gap shown: the density is the
// listener's own from then on, whatever theme they choose.
func (a *app) setSpectrumGap(bar bool, v float64) {
	s := &a.settings
	l := a.spectrumLayout()
	s.SpectrumDensity, s.SpectrumBars = densityCustom, l.Bars
	s.SpectrumBarGap, s.SpectrumSegGap = float64(l.BarGap), float64(l.SegGap)
	if bar {
		s.SpectrumBarGap = viz.ClampGap(v)
	} else {
		s.SpectrumSegGap = viz.ClampGap(v)
	}
}

// spectrumInactive returns how the segments not lit show: as the listener
// chose, or as the theme does.
func (a *app) spectrumInactive() string {
	switch m := a.settings.SpectrumInactive; m {
	case viz.InactiveOff, viz.InactiveSubtle, viz.InactiveVisible:
		return m
	}
	return viz.ThemeInactive(a.spectrumTheme())
}

// spectrumTarget returns the style the bars go to: the theme's, or the
// colors of the cover Now Playing shows, the last one's while the next
// loads.
func (a *app) spectrumTarget() viz.Style {
	switch id := a.spectrumTheme(); id {
	case viz.ThemeApple:
		return viz.AppleStyle(a.settings.SpectrumAppleColor)
	case viz.ThemeAlbum:
	default:
		return viz.StyleOf(id)
	}
	st := &a.spectrum
	if art := a.npArt; art != nil && !art.failed && art != st.albumArt {
		st.album, st.albumArt, st.albumSet = viz.AlbumStyle(art.pal), art, true
	} else if !st.albumSet {
		st.album, st.albumSet = viz.AlbumStyle(nil), true
	}
	return st.album
}

// spectrumPane is the spectrum, and the chips that choose its theme and
// how dense its bars are.
func (a *app) spectrumPane(c *ui.Context) {
	still := c.Preferences().ReduceMotion
	ui.Column(c.Key("spectrum-pane")).Grow(1).Gap(14).Children(func() {
		ui.Box(c.Key("spectrum")).Grow(1).Basis(0).MinHeight(160).Draw(func(p *ui.Painter, r ui.Rect) {
			a.drawSpectrum(p, r, still)
		})
		// The themes, then the densities, which go under them as a group in
		// a narrow window rather than past its edge, or split up.
		theme, density := a.spectrumTheme(), a.spectrumDensity()
		ui.Row(c.Key("spectrum-chips")).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Row(c.Key("spectrum-theme-chips")).Gap(6).AlignItems(ui.Center).Children(func() {
				for _, t := range spectrumThemes {
					if a.spectrumChip(c, "theme-"+t.id, T(t.label), theme == t.id) && a.settings.SpectrumTheme != t.id {
						a.settings.SpectrumTheme = t.id
						a.settings.save()
						a.redraw()
					}
				}
			})
			ui.Spacer(c)
			ui.Row(c.Key("spectrum-density-chips")).Gap(6).AlignItems(ui.Center).Children(func() {
				for _, d := range spectrumDensities[1:4] {
					if a.spectrumChip(c, "density-"+d.id, T(d.label), density == d.id) && a.settings.SpectrumDensity != d.id {
						a.setSpectrumDensity(d.id)
						a.settings.save()
						a.redraw()
					}
				}
			})
		})
	})
}

// spectrumChip is a chip under the spectrum, reporting a click.
func (a *app) spectrumChip(c *ui.Context, key, label string, on bool) bool {
	b := ui.ButtonBase(c.Key(key)).Padding(5, 12).Radius(999).FocusRing(false).Label(label)
	bg, fg := ui.Transparent, white.Alpha(0.55)
	if on {
		bg, fg = white.Alpha(0.14), white
	} else if b.Hovered() {
		bg = white.Alpha(0.06)
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
	b.Children(func() { ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(fg) })
	return b.Clicked()
}

// spectrumSettings are the rows of the visualizer's settings about how the
// bars look, after its theme and frame rate.
func (a *app) spectrumSettings(c *ui.Context, row settingsRow) {
	s, st := &a.settings, &a.spectrum
	pick := func(key, label string, opts []struct{ id, label string }, id string, set func(string)) {
		labels := make([]string, len(opts))
		at := 0
		for i, o := range opts {
			labels[i] = o.label
			if o.id == id {
				at = i
			}
		}
		if i, ok := choice(c, key, label, labels, at); ok {
			set(opts[i].id)
			a.changed()
		}
	}
	row(T("Spectrum Density"), T("How many bars there are, and how far apart. Theme Default follows each theme; any other stays as themes change."), func() {
		id := s.SpectrumDensity
		if id == densityCustom && s.SpectrumBars == 0 {
			id = ""
		}
		pick("spectrum-density", T("Spectrum Density"), spectrumDensities, id, a.setSpectrumDensity)
	})
	if !st.gapsShown {
		l := a.spectrumLayout()
		st.barGap, st.segGap, st.gapsShown = float64(l.BarGap), float64(l.SegGap), true
	}
	gap := func(key, label string, v *float64, bar bool) {
		if ui.Slider(c.Key(key), v, viz.MinGap, viz.MaxGap).Step(0.01).Width(170).Label(label).Changed() {
			a.setSpectrumGap(bar, *v)
			a.changed()
			return
		}
		// Between moves, what the density has now, chosen elsewhere.
		if l := a.spectrumLayout(); bar {
			*v = float64(l.BarGap)
		} else {
			*v = float64(l.SegGap)
		}
	}
	row(T("Bar Gap"), T("The space between neighbouring bars. Moving it makes the density Custom."), func() {
		gap("spectrum-bar-gap", T("Bar Gap"), &st.barGap, true)
	})
	row(T("Segment Gap"), T("The space between the segments of a bar. Moving it makes the density Custom."), func() {
		gap("spectrum-seg-gap", T("Segment Gap"), &st.segGap, false)
	})
	row(T("Unlit Segments"), T("How the segments not lit show: not at all, faintly, or as a dark LED matrix. Theme Default follows each theme."), func() {
		pick("spectrum-inactive", T("Unlit Segments"), spectrumInactives, s.SpectrumInactive, func(id string) { s.SpectrumInactive = id })
	})
	row(T("Apple Color"), T("The colors of Apple Colors: one of Apple's own, or all of them."), func() {
		pick("spectrum-apple", T("Apple Color"), spectrumAppleColors, s.SpectrumAppleColor, func(id string) { s.SpectrumAppleColor = id })
	})
	row(T("Glow"), T("A faint glow around the top of the lit bars. Uses a little more power."), func() {
		if ui.Switch(c.Key("spectrum-glow"), &s.SpectrumGlow).Label(T("Glow")).Changed() {
			a.changed()
		}
	})
	row(T("Peak Hold"), T("A mark that holds a moment where each bar peaked, then falls."), func() {
		if ui.Switch(c.Key("spectrum-peaks"), &s.SpectrumPeaks).Label(T("Peak Hold")).Changed() {
			a.changed()
		}
	})
}

// drawSpectrum asks the runner for the spectrum of what plays and draws it
// in r, asking for frames as often as the bars move, and none once they
// rest with nothing playing.
func (a *app) drawSpectrum(p *ui.Painter, r ui.Rect, still bool) {
	st := &a.spectrum
	now := p.Now()
	fps := a.spectrumRate(still)
	l := a.spectrumLayout()
	var h viz.Playhead
	f := st.shown
	if f == nil {
		if st.run == nil || st.fps != fps || st.bars != l.Bars {
			// Another rate or count of bars is another runner: the last
			// one stops, unasked.
			pl := a.pl
			st.run, st.fps, st.bars = viz.NewRunner(viz.Config{Bars: l.Bars}, fps), fps, l.Bars
			st.run.OnStop = func() { pl.SetTapping(false) }
		}
		// Every frame, as a runner that stopped turned it off.
		a.pl.SetTapping(true)
		if ph, ok := a.pl.Playhead(); ok {
			h = viz.Playhead{Frame: ph.Frame, At: ph.At, Playing: ph.Playing, Gain: ph.Gain, Song: ph.Song}
			if ph.Tap != nil {
				h.Source = ph.Tap
			}
		}
		st.run.Ask(h)
		st.run.Read(&st.frame)
		f = &st.frame
	}

	target := a.spectrumTarget()
	if !st.started {
		st.from, st.to, st.started = target, target, true
	}
	if target != st.to {
		st.from, st.to, st.fadeAt = st.to, target, now
	}
	k := float32(1)
	if !still && now.Before(st.fadeAt.Add(spectrumFade)) {
		k = clamp01(float32(now.Sub(st.fadeAt)) / float32(spectrumFade))
		k = 1 - (1-k)*(1-k)
	}

	p.Fill(r, ui.RGBA(4, 6, 10, 0.55), 18)
	// Room about the bars, more of it in a bigger pane.
	pad := min(max(min(r.W, r.H)*0.055, 16), 30)
	g, ok := viz.NewGrid(r.X+pad, r.Y+pad, r.W-2*pad, r.H-2*pad, p.Scale(), l)
	if !ok {
		return
	}
	segs, mode := g.Segs, a.spectrumInactive()
	for i := range segs {
		t := float32(i) / float32(segs-1)
		st.cols[i], st.offs[i] = st.to.At(t), st.to.OffAt(t, mode)
		if k < 1 {
			st.cols[i] = st.from.At(t).Mix(st.cols[i], k)
			st.offs[i] = st.from.OffAt(t, mode).Mix(st.offs[i], k)
		}
	}
	glow := st.to.Glow
	if k < 1 {
		glow = st.from.Glow + (glow-st.from.Glow)*k
	}
	if !a.settings.SpectrumGlow || !p.Accelerated() {
		glow = 0 // a blur a bar is too much for the CPU
	}
	peaks := a.settings.SpectrumPeaks && !still
	_, bw0 := g.Bar(0)
	_, sh0 := g.Seg(0)
	rad := min(2.5, min(bw0, sh0)*0.25)
	// The bars measured, over the bars there is room for: each shows the
	// loudest of those it stands for.
	m := min(f.Bars, viz.MaxBars)
	for b := range g.Bars {
		var lv, pk float32
		if m > 0 {
			lo := b * m / g.Bars
			for j := lo; j < max((b+1)*m/g.Bars, lo+1); j++ {
				lv, pk = max(lv, f.Levels[j]), max(pk, f.Peaks[j])
			}
		}
		x, bw := g.Bar(b)
		lit := min(segs, int(lv*float32(segs)+0.5))
		if glow > 0 && lit > 0 {
			// About the top few segments lit, not the whole bar.
			top, _ := g.Seg(lit - 1)
			y, sh := g.Seg(max(0, lit-3))
			p.Shadow(ui.Rect{X: x, Y: top, W: bw, H: y + sh - top}, rad, 0, 0, 8, 0, st.cols[lit-1].Alpha(glow*0.4))
		}
		for i := range segs {
			y, sh := g.Seg(i)
			seg := ui.Rect{X: x, Y: y, W: bw, H: sh}
			switch {
			case i < lit:
				p.Fill(seg, st.cols[i], rad)
			case st.offs[i].A > 0:
				p.Fill(seg, st.offs[i], rad)
			}
		}
		if pi := int(pk*float32(segs)+0.5) - 1; peaks && pi >= lit && pi < segs && pk > 0.02 {
			t := float32(pi) / float32(segs-1)
			c := st.to.PeakAt(t)
			if k < 1 {
				c = st.from.PeakAt(t).Mix(c, k)
			}
			y, sh := g.Seg(pi)
			ph := max(1/p.Scale(), sh*0.5)
			p.Fill(ui.Rect{X: x, Y: y, W: bw, H: ph}, c, min(rad, ph/2))
		}
	}
	switch {
	case st.shown != nil:
	case !f.Resting || h.Playing:
		p.AnimationFrameRate(float32(fps))
	case k < 1:
		p.AnimationFrame()
	}
}
