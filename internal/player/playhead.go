package player

import "time"

// SetTapping has the playing stream keep what it decodes for the spectrum,
// or stop: off, nothing is kept.
func (p *Player) SetTapping(on bool) { p.tapping.on.Store(on) }

// A Playhead is where what plays is, as the spectrum needs it: what the
// stream decoded, the frame heard at At, and how loud it is heard.
type Playhead struct {
	Tap     *Tap
	Frame   int64
	At      time.Time
	Playing bool
	Gain    float64 // the listener's volume, faded as playback pauses: 0 muted
	Song    int     // changes as another track loads
}

// Playhead returns where what plays is, or false while nothing plays here:
// loading, or playing in a remote source's own app.
func (p *Player) Playhead() (Playhead, bool) {
	if p.watch == nil || p.out == nil || p.stream == nil || p.remote != nil {
		return Playhead{}, false
	}
	rate := float64(p.stream.SampleRate())
	return Playhead{
		Tap:     p.watch.tap,
		Frame:   int64(p.Position().Seconds() * rate),
		At:      p.now(),
		Playing: p.playing && !p.seeking,
		Gain:    p.volume * p.volume * p.fade.current(),
		Song:    p.gen,
	}, true
}
