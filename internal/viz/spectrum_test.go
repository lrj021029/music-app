package viz

import (
	"math"
	"strconv"
	"testing"
	"time"
)

const frame = time.Second / 30

// tones fills s with sines of hz, each at amplitude amp.
func tones(s *Spectrum, amp float64, hz ...float64) {
	for i := range s.Samples {
		var v float64
		for _, f := range hz {
			v += amp * math.Sin(2*math.Pi*f*float64(i)/float64(s.rate))
		}
		s.Samples[i] = float32(v)
	}
}

// barOf returns the bar hz is in.
func barOf(s *Spectrum, hz float64) int {
	for b := range s.bars {
		if hz >= s.band[b].loHz && hz < s.band[b].hiHz {
			return b
		}
	}
	return -1
}

func loudest(s *Spectrum) int {
	top := 0
	for b := range s.bars {
		if s.level[b] > s.level[top] {
			top = b
		}
	}
	return top
}

// A tone lifts the bar its pitch is in above the rest, low, middle and high
// alike.
func TestToneLiftsItsBar(t *testing.T) {
	for _, hz := range []float64{64, 420, 2700, 8000} {
		s := NewSpectrum(Config{}, 44100)
		tones(s, 0.5, hz)
		for range 10 {
			s.Step(frame, true, 1)
		}
		if got, want := loudest(s), barOf(s, hz); got != want {
			t.Errorf("%v Hz lifted bar %d, not %d", hz, got, want)
		}
	}
}

// The bars rise at once with the music, fall away slowly once it stops, the
// peaks holding a moment above them, and then all come to rest.
func TestBarsRiseFastFallSlowly(t *testing.T) {
	s := NewSpectrum(Config{}, 44100)
	tones(s, 0.5, 420)
	for range 3 {
		s.Step(frame, true, 1)
	}
	b := loudest(s)
	top := s.level[b]
	if top < 0.8 {
		t.Fatalf("rose to %v in a tenth of a second", top)
	}
	for range 9 {
		s.Step(frame, false, 1)
	}
	if s.level[b] < 0.05 || s.level[b] > top/2 {
		t.Errorf("fell from %v to %v in 0.3 s", top, s.level[b])
	}
	if s.peak[b] != top {
		t.Errorf("peak fell from %v to %v while it holds", top, s.peak[b])
	}
	for range 60 {
		s.Step(frame, false, 1)
	}
	if !s.Resting() {
		t.Errorf("not at rest 2 s after the music stopped: %v, peak %v", s.level[b], s.peak[b])
	}
}

// The bass falls away slower than the highs, as a kick's swell outlasts a
// cymbal's tick.
func TestBassFallsSlowerThanHighs(t *testing.T) {
	s := NewSpectrum(Config{}, 44100)
	tones(s, 0.25, 80, 8000)
	for range 10 {
		s.Step(frame, true, 1)
	}
	low, high := barOf(s, 80), barOf(s, 8000)
	low0, high0 := s.level[low], s.level[high]
	for range 4 {
		s.Step(frame, false, 1)
	}
	if s.level[low]/low0 <= s.level[high]/high0 {
		t.Errorf("bass kept %v of itself, the highs %v", s.level[low]/low0, s.level[high]/high0)
	}
}

// The bars follow what is heard: lower turned down, and at rest muted.
func TestBarsFollowVolume(t *testing.T) {
	at := func(gain float64) (*Spectrum, float32) {
		s := NewSpectrum(Config{}, 44100)
		tones(s, 0.5, 420)
		for range 10 {
			s.Step(frame, true, gain)
		}
		return s, s.level[barOf(s, 420)]
	}
	_, full := at(1)
	_, quiet := at(0.1)
	if quiet >= full-0.2 {
		t.Errorf("at a tenth of the volume %v, at all of it %v", quiet, full)
	}
	if s, _ := at(0); !s.Resting() {
		t.Error("muted, the bars move")
	}
}

// Once another song plays, the last song's bars and peaks are gone within a
// few frames: none of it lingers over the new one. A peak used to hold where
// the old song left it.
func TestChangedClearsTheLastSong(t *testing.T) {
	s := NewSpectrum(Config{}, 44100)
	tones(s, 0.5, 420, 2700)
	for range 10 {
		s.Step(frame, true, 1)
	}
	s.Changed()
	for range 8 {
		s.Step(frame, false, 1)
	}
	if !s.Resting() {
		t.Errorf("the last song still shows: %v, peaks %v", s.Levels(), s.Peaks())
	}
}

// Measuring makes no garbage, frame after frame.
func TestStepAllocatesNothing(t *testing.T) {
	s := NewSpectrum(Config{}, 44100)
	tones(s, 0.5, 420)
	if n := testing.AllocsPerRun(50, func() { s.Step(frame, true, 0.8) }); n != 0 {
		t.Errorf("%v allocations a step", n)
	}
}

func BenchmarkStep(b *testing.B) {
	for _, size := range []int{2048, 4096} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			s := NewSpectrum(Config{FFTSize: size}, 44100)
			tones(s, 0.5, 420)
			for b.Loop() {
				s.Step(frame, true, 1)
			}
		})
	}
}
