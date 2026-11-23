package player

import (
	"math"
	"sync/atomic"
)

// A Tap keeps the last couple of seconds of what the playing stream decoded,
// mixed down to mono, for the spectrum to look at as it is heard. The stream
// writes it on package audio's goroutine, which decodes a second or so ahead
// of the output, and never waits for a reader: one too slow finds the frames
// it read written over, and is told so.
type Tap struct {
	rate int
	on   *tapSwitch
	ring atomic.Pointer[tapRing]
	// moves counts the stream's seeks, twice each: odd while one lands.
	moves atomic.Uint64
	// Frames in [start, end) are kept, unless written over since: start is
	// where the stream last moved to, or where keeping last began.
	start, end atomic.Int64
	// head is the frame written up to, set before the samples are: frames
	// a ring's length below it may be being written over.
	head atomic.Int64
	pos  int64 // the frame the stream decodes next: its goroutine's alone
}

// A tapSwitch says whether taps keep what they are given: something shows
// the spectrum.
type tapSwitch struct{ on atomic.Bool }

type tapRing struct {
	s    []atomic.Uint32 // the sample of frame f, as float32 bits, at f&mask
	mask int64
}

// tapSeconds is how much a tap keeps: the second package audio decodes
// ahead, the longest window measured, and room to read it in.
const tapSeconds = 2

func newTap(rate int, on *tapSwitch) *Tap { return &Tap{rate: rate, on: on} }

// Rate is the rate of the frames kept.
func (t *Tap) Rate() int { return t.rate }

// write keeps samples, interleaved of channels, decoded from where the
// stream was. Off, it only follows where the stream is.
func (t *Tap) write(samples []float32, channels int) {
	if t == nil || channels < 1 {
		return
	}
	at, frames := t.pos, int64(len(samples)/channels)
	t.pos += frames
	if !t.on.on.Load() {
		t.start.Store(t.pos)
		t.end.Store(t.pos)
		t.head.Store(t.pos)
		return
	}
	r := t.ring.Load()
	if r == nil {
		n := int64(1)
		for n < int64(t.rate)*tapSeconds {
			n <<= 1
		}
		r = &tapRing{s: make([]atomic.Uint32, n), mask: n - 1}
		t.ring.Store(r)
	}
	t.head.Store(t.pos)
	for i := range frames {
		j := int(i) * channels
		v := samples[j]
		if channels > 1 {
			v = (v + samples[j+1]) / 2
		}
		r.s[(at+i)&r.mask].Store(math.Float32bits(v))
	}
	t.end.Store(t.pos)
}

// seek has the tap follow the stream to frame, dropping what it kept.
func (t *Tap) seek(frame int64) {
	if t == nil {
		return
	}
	t.moves.Add(1)
	t.pos = frame
	t.start.Store(frame)
	t.end.Store(frame)
	t.head.Store(frame)
	t.moves.Add(1)
}

// Window copies the len(dst) frames before end into dst, and reports
// whether they were all there: decoded since the stream last moved, and not
// written over before they were read.
func (t *Tap) Window(end int64, dst []float32) bool {
	if t == nil {
		return false
	}
	r := t.ring.Load()
	n := int64(len(dst))
	if r == nil || n > int64(len(r.s))/2 {
		return false
	}
	m, from := t.moves.Load(), end-n
	if m&1 != 0 || from < t.start.Load() || end > t.end.Load() {
		return false
	}
	for i := range dst {
		dst[i] = math.Float32frombits(r.s[(from+int64(i))&r.mask].Load())
	}
	return t.moves.Load() == m && from >= t.head.Load()-int64(len(r.s))
}
