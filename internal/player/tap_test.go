package player

import (
	"sync/atomic"
	"testing"
)

// ramp is n frames of stereo, frame i at i in one channel and i+2 in the
// other: i+1 mixed down.
func ramp(n int) []float32 {
	s := make([]float32, 2*n)
	for i := range n {
		s[2*i], s[2*i+1] = float32(i), float32(i)+2
	}
	return s
}

func tapOn() *tapSwitch {
	var on tapSwitch
	on.on.Store(true)
	return &on
}

// A tap gives back the frames the stream decoded, mixed down to mono, and
// none it has yet to decode or has since written over.
func TestTapWindow(t *testing.T) {
	tp := newTap(1000, tapOn()) // keeps 2048 frames
	s := ramp(3000)
	tp.write(s[:2*1500], 2)
	dst := make([]float32, 512)
	if !tp.Window(1500, dst) || dst[0] != 989 || dst[511] != 1500 {
		t.Fatalf("frames 988 to 1500 are %v to %v", dst[0], dst[511])
	}
	if tp.Window(1600, dst) {
		t.Error("gave frames not yet decoded")
	}
	tp.write(s[2*1500:], 2)
	if !tp.Window(3000, dst) || dst[511] != 3000 {
		t.Errorf("the last frames are %v", dst[511])
	}
	if tp.Window(800, dst) {
		t.Error("gave frames written over")
	}
}

// A seek drops what the tap kept: the frames before it are of where the
// song was, not where it plays.
func TestTapSeekDrops(t *testing.T) {
	tp := newTap(1000, tapOn())
	tp.write(ramp(1500), 2)
	tp.seek(5000)
	dst := make([]float32, 512)
	if tp.Window(1500, dst) {
		t.Error("gave frames from before the seek")
	}
	tp.write(ramp(600), 2)
	if !tp.Window(5600, dst) || dst[0] != 89 {
		t.Errorf("frames from the seek on start at %v", dst[0])
	}
	if tp.Window(5100, dst) {
		t.Error("gave frames from before where it landed")
	}
}

// Off, a tap keeps nothing; turned on, it keeps what comes after.
func TestTapOff(t *testing.T) {
	var on tapSwitch
	tp := newTap(1000, &on)
	tp.write(ramp(1000), 2)
	dst := make([]float32, 256)
	if tp.Window(1000, dst) || tp.ring.Load() != nil {
		t.Fatal("off, it kept frames")
	}
	on.on.Store(true)
	tp.write(ramp(300), 2)
	if !tp.Window(1300, dst) || dst[0] != 45 {
		t.Errorf("frames after it was turned on start at %v", dst[0])
	}
	if tp.Window(1100, dst) {
		t.Error("gave frames from while it was off")
	}
}

// A reader never takes frames torn by the stream writing over them, or
// seeking, as it reads: what it is given is all what was decoded there.
func TestTapReadWhileWriting(t *testing.T) {
	tp := newTap(1000, tapOn())
	var done atomic.Bool
	go func() {
		defer done.Store(true)
		buf := make([]float32, 2*64)
		// Frame f is f, then once it seeks back, f and a quarter.
		for _, run := range []struct {
			from, to int64
			mark     float32
		}{{0, 100000, 0}, {1000, 100000, 0.25}} {
			tp.seek(run.from)
			for f := run.from; f < run.to; f += 64 {
				for i := range int64(64) {
					v := float32(f+i) + run.mark
					buf[2*i], buf[2*i+1] = v, v
				}
				tp.write(buf, 2)
			}
		}
	}()
	dst := make([]float32, 512)
	read := 0
	for last := false; !last; {
		last = done.Load()
		end := tp.end.Load()
		if !tp.Window(end, dst) {
			continue
		}
		read++
		mark := dst[0] - float32(end-512)
		for i, v := range dst {
			if v-float32(end-512+int64(i)) != mark || mark != 0 && mark != 0.25 {
				t.Fatalf("frame %d of a window ending at %d is %v", i, end, v)
			}
		}
	}
	if read == 0 {
		t.Error("read nothing")
	}
}
