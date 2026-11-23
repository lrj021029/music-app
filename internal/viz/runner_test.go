package viz

import (
	"math"
	"sync/atomic"
	"testing"
	"time"
)

// sine is a tone that plays on forever.
type sine struct{ hz float64 }

func (s sine) Rate() int { return 44100 }

func (s sine) Window(end int64, dst []float32) bool {
	from := end - int64(len(dst))
	for i := range dst {
		dst[i] = float32(0.5 * math.Sin(2*math.Pi*s.hz*float64(from+int64(i))/44100))
	}
	return true
}

// Asked for, a runner measures what plays; no longer asked, its goroutine
// stops, and it starts again once asked again.
func TestRunnerRunsWhileAsked(t *testing.T) {
	var clock atomic.Int64
	clock.Store(time.Unix(1000, 0).UnixNano())
	ticks := make(chan time.Time)
	r := NewRunner(Config{}, 30)
	r.now = func() time.Time { return time.Unix(0, clock.Load()) }
	r.tick = func(time.Duration) (<-chan time.Time, func()) { return ticks, func() {} }
	stopped := func() {
		t.Helper()
		clock.Add(int64(time.Second))
		ticks <- time.Time{}
		for deadline := time.Now().Add(2 * time.Second); r.Running(); time.Sleep(time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatal("it measures on, no longer asked")
			}
		}
	}

	h := Playhead{Source: sine{420}, At: r.now(), Playing: true, Gain: 1}
	r.Ask(h)
	for range 6 {
		clock.Add(int64(frame))
		ticks <- time.Time{}
	}
	ticks <- time.Time{} // the last frame measured
	var f Frame
	r.Read(&f)
	if f.Resting || f.Bars != 32 {
		t.Fatalf("measured %d bars of a tone playing, resting %v", f.Bars, f.Resting)
	}
	stopped()
	r.Ask(h)
	if !r.Running() {
		t.Fatal("asked again, it doesn't measure")
	}
	stopped()
}
