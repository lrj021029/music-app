// Package viz measures the spectrum of what plays, for the visualizer: bars
// spaced evenly in pitch, rising with the music at once and falling away
// slowly, each with a peak that holds a moment before it falls.
package viz

import (
	"cmp"
	"math"
	"time"
)

// MaxBars is the most bars a spectrum has.
const MaxBars = 64

// Config is how a spectrum measures.
type Config struct {
	Bars    int     // 32 if 0, from 8 to MaxBars
	FFTSize int     // the window at 48 kHz and below, 2048 if 0: twice it at 96 kHz, and on
	MinHz   float64 // the lowest pitch shown, 40 Hz if 0
	MaxHz   float64 // the highest, 16 kHz if 0
}

func (c Config) withDefaults() Config {
	c.Bars = min(max(cmp.Or(c.Bars, 32), 8), MaxBars)
	c.FFTSize = cmp.Or(c.FFTSize, 2048)
	c.MinHz = cmp.Or(c.MinHz, 40)
	c.MaxHz = cmp.Or(c.MaxHz, 16000)
	return c
}

const (
	// spanDB is how far below the top the bars reach down, and headroomDB
	// how far above the loudest bar lately the top is.
	spanDB     = 54
	headroomDB = 3
	// The top follows the music's loudness, up at once and down by agcFall
	// decibels a second, so that a quiet song fills the bars as a loud one
	// does; but never lower than agcFloor, where it would show only noise.
	agcFall  = 3
	agcFloor = -42
	agcCeil  = 12
	agcStart = -18
	// tiltDB lifts each octave above 1 kHz, and lowers each below: music has
	// less in its highs than its lows, as pink noise does.
	tiltDB = 4.5
	// A peak holds peakHold seconds, then falls faster and faster.
	peakHold    = 0.5
	peakGravity = 2.5
	// For flushFor seconds past a change of song the bars fall in flushTau,
	// rather than in their own time, so that none of the last lingers.
	flushFor = 0.25
	flushTau = 0.035
	// restBelow is how low a bar is at rest.
	restBelow = 0.002
)

// response is how fast a bar at hz rises and falls, in seconds: the bass's
// kick swells and fades, the voice moves quicker, cymbals quickest.
func response(hz float64) (attack, release float64) {
	switch {
	case hz < 250:
		return 0.020, 0.260
	case hz < 4000:
		return 0.015, 0.180
	}
	return 0.010, 0.120
}

// A Spectrum is bars measured from windows of samples, and moved on in
// time. It allocates as it is made, and not after: Step makes no garbage.
type Spectrum struct {
	// Samples is the window to measure next, of what plays, in mono.
	Samples []float32

	bars, size, rate int
	fft              *fft
	window, power    []float32
	band             [MaxBars]band
	attack, release  [MaxBars]float64

	level, peak, fall [MaxBars]float32
	hold              [MaxBars]float64 // the seconds a peak still holds
	ref               float32          // the loudest bar lately, in decibels
	flush             float64          // the seconds of the fast fall left
}

// band is the bins a bar takes the mean power of; or, narrower than a bin,
// the point it takes the power at, between two.
type band struct {
	lo, hi     int
	at         float32
	narrow     bool
	tilt       float32
	loHz, hiHz float64
}

// NewSpectrum makes a spectrum of what plays at rate.
func NewSpectrum(cfg Config, rate int) *Spectrum {
	cfg = cfg.withDefaults()
	size := 1
	for size < cfg.FFTSize {
		size <<= 1
	}
	// The same span of time at higher rates, and so the same resolution.
	for r := rate; r > 50000 && size < 16384; r /= 2 {
		size <<= 1
	}
	s := &Spectrum{Samples: make([]float32, size), bars: cfg.Bars, size: size, rate: rate, fft: newFFT(size), window: make([]float32, size), power: make([]float32, size/2+1), ref: agcStart}
	for i := range s.window {
		s.window[i] = float32(0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(size)))
	}
	maxHz := min(cfg.MaxHz, float64(rate)/2*0.95)
	binHz := float64(rate) / float64(size)
	lo, hi := math.Log(cfg.MinHz), math.Log(maxHz)
	for b := range s.bars {
		f0 := math.Exp(lo + (hi-lo)*float64(b)/float64(s.bars))
		f1 := math.Exp(lo + (hi-lo)*float64(b+1)/float64(s.bars))
		fc := math.Sqrt(f0 * f1)
		bd := band{lo: int(math.Ceil(f0 / binHz)), hi: int(math.Floor(f1 / binHz)), at: float32(fc / binHz), tilt: float32(tiltDB * math.Log2(fc/1000)), loHz: f0, hiHz: f1}
		// No bin's middle in it: it takes the power between the two nearest.
		bd.narrow = bd.hi < bd.lo
		bd.lo, bd.hi = max(1, bd.lo), min(size/2, bd.hi)
		s.band[b] = bd
		s.attack[b], s.release[b] = response(fc)
	}
	return s
}

// Bars returns how many bars there are.
func (s *Spectrum) Bars() int { return s.bars }

// Levels returns the bars' heights, from 0 to 1.
func (s *Spectrum) Levels() []float32 { return s.level[:s.bars] }

// Peaks returns the bars' peaks, from 0 to 1.
func (s *Spectrum) Peaks() []float32 { return s.peak[:s.bars] }

// Resting reports whether every bar and peak is down at rest, so that
// nothing moves until the music does.
func (s *Spectrum) Resting() bool {
	for b := range s.bars {
		if s.level[b] >= restBelow || s.peak[b] >= restBelow {
			return false
		}
	}
	return true
}

// Changed says another song plays: the bars fall at once, their peaks with
// them, and the top starts over, nearer where songs start.
func (s *Spectrum) Changed() {
	s.flush = flushFor
	s.ref = (s.ref + agcStart) / 2
}

// Step moves the bars on by dt: towards what Samples holds, if live, or down
// to rest. gain is how loud what plays is heard, from 0 for nothing to 1.
func (s *Spectrum) Step(dt time.Duration, live bool, gain float64) {
	sec := dt.Seconds()
	var target [MaxBars]float32
	if live && gain > 0 {
		s.measure(&target, sec, gain)
	}
	s.move(&target, sec)
}

func (s *Spectrum) measure(target *[MaxBars]float32, sec, gain float64) {
	f := s.fft
	for i, v := range s.Samples {
		f.re[i], f.im[i] = v*s.window[i], 0
	}
	f.transform()
	for k := range s.power {
		s.power[k] = f.re[k]*f.re[k] + f.im[k]*f.im[k]
	}
	// A full-scale sine is 0 dB in its bin.
	norm := float64(s.size) / 4
	norm *= norm
	loudest := float32(-200)
	for b := range s.bars {
		bd := &s.band[b]
		var p float32
		if bd.narrow {
			k := int(bd.at)
			t := bd.at - float32(k)
			p = s.power[k]*(1-t) + s.power[min(k+1, len(s.power)-1)]*t
		} else {
			for k := bd.lo; k <= bd.hi; k++ {
				p += s.power[k]
			}
			p /= float32(bd.hi - bd.lo + 1)
		}
		db := float32(10*math.Log10(float64(p)/norm+1e-12)) + bd.tilt
		target[b] = db
		loudest = max(loudest, db)
	}
	s.ref = min(max(loudest, s.ref-agcFall*float32(sec), agcFloor), agcCeil)
	// What is heard more quietly shows lower: the top stays where the music's
	// loudness put it.
	bottom := s.ref + headroomDB - spanDB - float32(20*math.Log10(gain))
	for b := range s.bars {
		target[b] = min(max((target[b]-bottom)/spanDB, 0), 1)
	}
}

func (s *Spectrum) move(target *[MaxBars]float32, sec float64) {
	flushing := s.flush > 0
	for b := range s.bars {
		tau := s.release[b]
		if target[b] > s.level[b] {
			tau = s.attack[b]
		} else if flushing {
			tau = flushTau
		}
		s.level[b] += (target[b] - s.level[b]) * float32(1-math.Exp(-sec/tau))
		switch {
		case flushing:
			s.peak[b], s.hold[b], s.fall[b] = s.level[b], 0, 0
		case s.level[b] >= s.peak[b]:
			s.peak[b], s.hold[b], s.fall[b] = s.level[b], peakHold, 0
		case s.hold[b] > 0:
			s.hold[b] -= sec
		default:
			s.fall[b] += float32(peakGravity * sec)
			s.peak[b] = max(s.level[b], s.peak[b]-s.fall[b]*float32(sec))
		}
	}
	s.flush = max(0, s.flush-sec)
}
