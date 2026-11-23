package viz

import (
	"sync"
	"sync/atomic"
	"time"
)

// A Source is what plays, as a spectrum reads it: a player's tap.
type Source interface {
	Rate() int
	// Window copies the len(dst) frames before end into dst, and reports
	// whether it had them.
	Window(end int64, dst []float32) bool
}

// A Playhead is where playback is, as the main thread last saw it.
type Playhead struct {
	Source  Source
	Frame   int64     // the frame heard at At
	At      time.Time //
	Playing bool
	Gain    float64 // how loud it is heard, from 0 to 1
	Song    int     // changes with the song
}

// A Frame is the spectrum measured last.
type Frame struct {
	Bars          int
	Levels, Peaks [MaxBars]float32
	Resting       bool // nothing moves until the music does
}

// demandGrace is how long a runner goes on measuring, not asked: what shows
// the spectrum asks as it draws, so this is a few frames of its missing.
const demandGrace = 250 * time.Millisecond

// A Runner measures a spectrum on a goroutine of its own, fps times a
// second, for as long as something asks for it. Not asked, the goroutine
// stops, and starts again as it is asked; the audio's goroutine never waits
// on it, nor it on the audio's.
type Runner struct {
	cfg  Config
	fps  int
	now  func() time.Time
	tick func(time.Duration) (<-chan time.Time, func())

	// OnStop, if set before the runner is first asked, runs on its
	// goroutine as it stops, no longer asked: what feeds it may stop too.
	OnStop func()

	asked   atomic.Int64 // when it was last asked for, in Unix nanoseconds
	running atomic.Bool

	mu   sync.Mutex // guards head and out
	head Playhead
	out  Frame

	// The goroutine's alone:
	spec *Spectrum
	rate int
	song int
	last time.Time
}

// NewRunner makes a runner that measures as cfg says, fps times a second.
func NewRunner(cfg Config, fps int) *Runner {
	r := &Runner{cfg: cfg.withDefaults(), fps: min(max(fps, 10), 120), now: time.Now, tick: ticker}
	r.out.Resting = true
	return r
}

func ticker(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}

// Ask tells the runner where playback is, and that something shows the
// spectrum: it measures until it is no longer asked.
func (r *Runner) Ask(h Playhead) {
	r.mu.Lock()
	r.head = h
	r.mu.Unlock()
	r.asked.Store(r.now().UnixNano())
	if r.running.CompareAndSwap(false, true) {
		go r.loop()
	}
}

// Read copies the spectrum measured last into f.
func (r *Runner) Read(f *Frame) {
	r.mu.Lock()
	*f = r.out
	r.mu.Unlock()
}

// Running reports whether the runner's goroutine measures.
func (r *Runner) Running() bool { return r.running.Load() }

func (r *Runner) wanted(now time.Time) bool {
	return now.UnixNano()-r.asked.Load() <= int64(demandGrace)
}

func (r *Runner) loop() {
	ticks, stop := r.tick(time.Second / time.Duration(r.fps))
	defer stop()
	for range ticks {
		now := r.now()
		if !r.wanted(now) {
			r.running.Store(false)
			// Asked again as it stopped, it goes on.
			if !r.wanted(r.now()) {
				if r.OnStop != nil {
					r.OnStop()
				}
				return
			}
			if !r.running.CompareAndSwap(false, true) {
				return
			}
		}
		r.mu.Lock()
		h := r.head
		r.mu.Unlock()
		r.step(h, now)
	}
}

func (r *Runner) step(h Playhead, now time.Time) {
	dt := time.Second / time.Duration(r.fps)
	if !r.last.IsZero() {
		dt = min(max(now.Sub(r.last), 0), 100*time.Millisecond)
	}
	r.last = now
	if h.Source != nil && (r.spec == nil || h.Source.Rate() != r.rate) {
		r.rate = h.Source.Rate()
		r.spec = NewSpectrum(r.cfg, r.rate)
	}
	if r.spec == nil {
		return
	}
	if h.Song != r.song {
		r.song = h.Song
		r.spec.Changed()
	}
	live := false
	if h.Playing && h.Source != nil {
		// The window centered on what is heard now.
		heard := h.Frame + int64(now.Sub(h.At).Seconds()*float64(r.rate))
		live = h.Source.Window(heard+int64(len(r.spec.Samples)/2), r.spec.Samples)
	}
	r.spec.Step(dt, live, h.Gain)
	r.mu.Lock()
	r.out.Bars = r.spec.bars
	r.out.Levels, r.out.Peaks = r.spec.level, r.spec.peak
	r.out.Resting = r.spec.Resting()
	r.mu.Unlock()
}
