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

const (
	// spectrumFade is how long the bars take to go from one style's
	// colors to the next: another theme, or another cover's.
	spectrumFade = 600 * time.Millisecond
	// spectrumStillRate is how often they move with Reduce Motion on,
	// their peaks left out.
	spectrumStillRate = 12
	spectrumMaxSegs   = 48
)

// spectrumState is the spectrum as it was last drawn.
type spectrumState struct {
	run   *viz.Runner
	fps   int // the runner's
	frame viz.Frame
	// The style the bars fade from, and to, since fadeAt.
	from, to viz.Style
	fadeAt   time.Time
	started  bool
	// The cover's style, for Album Adaptive, and the cover it is of.
	album    viz.Style
	albumArt *artwork
	albumSet bool
	cols     [spectrumMaxSegs]ui.Color // the segments' colors, foot to top
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

// spectrumTarget returns the style the bars go to: the theme's, or the
// colors of the cover Now Playing shows, the last one's while the next
// loads.
func (a *app) spectrumTarget() viz.Style {
	id := a.spectrumTheme()
	if id != viz.ThemeAlbum {
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

// spectrumPane is the spectrum, and the chips that choose its theme.
func (a *app) spectrumPane(c *ui.Context) {
	still := c.Preferences().ReduceMotion
	ui.Column(c.Key("spectrum-pane")).Grow(1).Gap(14).Children(func() {
		ui.Box(c.Key("spectrum")).Grow(1).Basis(0).MinHeight(160).Draw(func(p *ui.Painter, r ui.Rect) {
			a.drawSpectrum(p, r, still)
		})
		ui.Row(c.Key("spectrum-themes")).Gap(6).AlignItems(ui.Center).Children(func() {
			for _, t := range spectrumThemes {
				a.spectrumChip(c, t.id, T(t.label))
			}
		})
	})
}

func (a *app) spectrumChip(c *ui.Context, id, label string) {
	b := ui.ButtonBase(c.Key("theme-"+id)).Padding(5, 12).Radius(999).FocusRing(false).Label(label)
	bg, fg := ui.Transparent, white.Alpha(0.55)
	if a.spectrumTheme() == id {
		bg, fg = white.Alpha(0.14), white
	} else if b.Hovered() {
		bg = white.Alpha(0.06)
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
	b.Children(func() { ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(fg) })
	if b.Clicked() && a.settings.SpectrumTheme != id {
		a.settings.SpectrumTheme = id
		a.settings.save()
		a.redraw()
	}
}

// drawSpectrum asks the runner for the spectrum of what plays and draws it
// in r, asking for frames as often as the bars move, and none once they
// rest with nothing playing.
func (a *app) drawSpectrum(p *ui.Painter, r ui.Rect, still bool) {
	st := &a.spectrum
	now := p.Now()
	fps := a.spectrumRate(still)
	if st.run == nil || st.fps != fps {
		// Another rate is another runner: the last one stops, unasked.
		pl := a.pl
		st.run, st.fps = viz.NewRunner(viz.Config{}, fps), fps
		st.run.OnStop = func() { pl.SetTapping(false) }
	}
	// Every frame, as a runner that stopped turned it off.
	a.pl.SetTapping(true)
	var h viz.Playhead
	if ph, ok := a.pl.Playhead(); ok {
		h = viz.Playhead{Frame: ph.Frame, At: ph.At, Playing: ph.Playing, Gain: ph.Gain, Song: ph.Song}
		if ph.Tap != nil {
			h.Source = ph.Tap
		}
	}
	st.run.Ask(h)
	st.run.Read(&st.frame)
	f := &st.frame

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
	in := ui.Rect{X: r.X + 22, Y: r.Y + 22, W: r.W - 44, H: r.H - 44}
	n := f.Bars
	if n == 0 {
		n = 32 // not measured yet: the bars at rest
	}
	if in.W < float32(n)*3 || in.H < 40 {
		return
	}
	pitch := in.W / float32(n)
	gap := max(2, pitch*0.26)
	bw := pitch - gap
	segs := min(spectrumMaxSegs, max(8, int(in.H/9)))
	sp := in.H / float32(segs)
	segGap := max(1.5, sp*0.28)
	sh := sp - segGap
	for i := range segs {
		t := float32(i) / float32(segs-1)
		st.cols[i] = st.to.At(t)
		if k < 1 {
			st.cols[i] = st.from.At(t).Mix(st.cols[i], k)
		}
	}
	off := st.to.Off
	glow := st.to.Glow
	if k < 1 {
		off = st.from.Off.Mix(off, k)
		glow = st.from.Glow + (glow-st.from.Glow)*k
	}
	if !p.Accelerated() {
		glow = 0 // a blur a bar is too much for the CPU
	}
	for b := range n {
		x := in.X + float32(b)*pitch + gap/2
		lv, pk := f.Levels[b], f.Peaks[b]
		lit := min(segs, int(lv*float32(segs)+0.5))
		if glow > 0 && lit > 0 {
			top := in.Y + in.H - float32(lit)*sp + segGap/2
			p.Shadow(ui.Rect{X: x, Y: top, W: bw, H: in.Y + in.H - top}, 2, 0, 0, 10, 0, st.cols[lit-1].Alpha(glow*0.45))
		}
		for i := range segs {
			seg := ui.Rect{X: x, Y: in.Y + in.H - float32(i+1)*sp + segGap/2, W: bw, H: sh}
			switch {
			case i < lit:
				p.Fill(seg, st.cols[i], 1.5)
			case off.A > 0:
				p.Fill(seg, off, 1.5)
			}
		}
		if pi := int(pk*float32(segs)+0.5) - 1; !still && pi >= lit && pi < segs && pk > 0.02 {
			t := float32(pi) / float32(segs-1)
			c := st.to.PeakAt(t)
			if k < 1 {
				c = st.from.PeakAt(t).Mix(c, k)
			}
			y := in.Y + in.H - float32(pi+1)*sp + segGap/2
			p.Fill(ui.Rect{X: x, Y: y, W: bw, H: max(1.5, sh*0.5)}, c, 1)
		}
	}
	switch {
	case !f.Resting || h.Playing:
		p.AnimationFrameRate(float32(fps))
	case k < 1:
		p.AnimationFrame()
	}
}
