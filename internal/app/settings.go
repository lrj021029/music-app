package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"

	"mygo-music/internal/i18n"
)

// settings are the listener's choices, kept in settings.json in the user
// data directory.
type settings struct {
	// Language is what the app speaks: i18n.English or i18n.Chinese, or
	// i18n.System for the system's.
	Language string `json:"language"`
	// Source is the ID of the source last used.
	Source string `json:"source,omitempty"`
	// Captions writes down episodes as they play, for their captions, on this
	// device, once the listener has the model.
	Captions bool `json:"captions"`
	// DisabledPlugins are the IDs of the plugins turned off. All are on until
	// then, the built-in one included.
	DisabledPlugins []string `json:"disabledPlugins,omitempty"`
	// PluginsChecked is when plugins were last checked for new versions, once a
	// day, and PluginUpdates the versions found, by plugin.
	PluginsChecked time.Time         `json:"pluginsChecked,omitzero"`
	PluginUpdates  map[string]string `json:"pluginUpdates,omitempty"`
	// Notch shows the quick player around the notch: the listener turns
	// it on.
	Notch bool `json:"notch"`
	// NotchActivity shows the cover and the music's levels beside the
	// notch while music plays, which hides the menu bar there.
	NotchActivity bool `json:"notchActivity"`
	// NotchVirtual keeps a notch of the app's own showing at the top of a
	// display without one; else it shows only while the pointer asks.
	NotchVirtual bool `json:"notchVirtual"`
	// NotchGreeted says the notch showed itself, once, as the first song
	// played (greet).
	NotchGreeted bool `json:"notchGreeted"`
	// Quality is what's asked of sources: standard, high or lossless. A source
	// may ignore it.
	Quality string `json:"quality"`
	// Appearance is light, dark, or system to follow the system.
	Appearance string `json:"appearance"`
	// GrowFromCover opens Now Playing out of the player bar's cover, and
	// closes it back into it: an experiment the listener turns on. Off, it
	// slides up from the bottom of the window.
	GrowFromCover bool `json:"growFromCover"`
	// Smoke makes the progress bars comets trailing smoke as music plays,
	// for a few percent of a core: the listener turns it on.
	Smoke bool `json:"smoke"`
	// LyricsStage shows lyrics on the stage, Folia's style, in Now
	// Playing, rather than scrolling.
	LyricsStage bool `json:"lyricsStage"`
	// StageStyle is how the stage sets its lyrics (stageStyles): empty for
	// Folia's classic, or the id of another of its visualizers.
	StageStyle string `json:"stageStyle,omitempty"`
	// PlayerStyle is how Now Playing shows the song playing
	// (playerStyles): empty for its cover, or a retro player's id.
	PlayerStyle string `json:"playerStyle,omitempty"`
	// StageSparks throws sparks off each word on the stage as it is sung,
	// whatever its style.
	StageSparks bool `json:"stageSparks,omitempty"`
	// Volume is where the volume was left, from 0 to 1.
	Volume float64 `json:"volume"`
	// Normalize evens out the loudness of tracks, by what their source
	// measured of each.
	Normalize bool `json:"normalize"`
	// HotComments shows a song's top comments in its quiet parts, in place of
	// the lyrics (quietComment), for a source with comments: the listener
	// turns it on.
	HotComments bool `json:"hotComments"`
	// Fog fills Now Playing's backdrop with fog of the cover's color,
	// rising and swelling with the bass: an experiment the listener turns
	// on.
	Fog bool `json:"fog"`
	// Spectrum shows the Spectrum tab in Now Playing, whose bars move with
	// what plays: off, nothing measures the music.
	Spectrum bool `json:"spectrum"`
	// SpectrumTheme is the colors of its bars (spectrumThemes): Classic
	// LED's if empty.
	SpectrumTheme string `json:"spectrumTheme,omitempty"`
	// SpectrumFPS is how often its bars move: 30 or 60.
	SpectrumFPS int `json:"spectrumFPS,omitempty"`
	// SpectrumDensity is how many bars there are and how far apart
	// (spectrumDensities), or "custom" for SpectrumBars and the gaps: the
	// theme's own if empty. Choosing a theme leaves it be.
	SpectrumDensity string `json:"spectrumDensity,omitempty"`
	// SpectrumBars, SpectrumBarGap and SpectrumSegGap are the custom
	// density's: the bars, and the space beside each bar and above each
	// segment, as a share of its room.
	SpectrumBars   int     `json:"spectrumBars,omitempty"`
	SpectrumBarGap float64 `json:"spectrumBarGap,omitempty"`
	SpectrumSegGap float64 `json:"spectrumSegGap,omitempty"`
	// SpectrumInactive is how the segments not lit show: "off", "subtle"
	// or "visible", the theme's way if empty.
	SpectrumInactive string `json:"spectrumInactive,omitempty"`
	// SpectrumGlow glows faintly around the top of the lit bars, and
	// SpectrumPeaks holds a mark a moment where each bar peaked.
	SpectrumGlow  bool `json:"spectrumGlow"`
	SpectrumPeaks bool `json:"spectrumPeaks"`
	// SpectrumAppleColor is the color of Apple Colors (viz.AppleStyle): all
	// of them if empty.
	SpectrumAppleColor string `json:"spectrumAppleColor,omitempty"`
	// AppIcon is where the app's icon shows on macOS (appIcons); elsewhere
	// it is always in the notification area.
	AppIcon string `json:"appIcon"`
	// CacheMB is how many megabytes the songs kept on disk may take; 0
	// keeps none.
	CacheMB int `json:"cacheMB"`
	// SearchHistory keeps submitted keywords, most recent first.
	SearchHistory []string `json:"searchHistory,omitempty"`

	path string // where they were loaded from, and are saved to
}

func defaultSettings() settings {
	return settings{Language: i18n.System, NotchActivity: true, Quality: "high", Appearance: "system", Volume: 0.8, Normalize: true, Spectrum: true, SpectrumTheme: "classic", SpectrumFPS: 30, SpectrumGlow: true, SpectrumPeaks: true, AppIcon: "dock-and-menubar", CacheMB: 2048}
}

func settingsPath() (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// loadSettings reads the settings, or returns the defaults.
func loadSettings() settings {
	s := defaultSettings()
	if path, err := settingsPath(); err == nil {
		if b, err := os.ReadFile(path); err == nil {
			json.Unmarshal(b, &s)
		}
		s.path = path
	}
	return s
}

// save keeps the settings where they were loaded from; defaults that were
// never loaded, as in tests, are not saved.
func (s settings) save() error {
	path := s.path
	if path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// applyAppearance makes the windows light, dark, or follow the system. Only
// a new one: it repaints every window, waiting for the main thread, and any
// setting changed applies it.
func (s settings) applyAppearance() {
	src := mygo.ThemeSystem
	switch s.Appearance {
	case "light":
		src = mygo.ThemeLight
	case "dark":
		src = mygo.ThemeDark
	}
	if src != mygo.Theme.Source() {
		mygo.Theme.SetSource(src)
	}
}

// qualityLevels are what can be asked of sources, by what they sound like.
var qualityLevels = []struct{ level, label string }{
	{"standard", N("Standard")},
	{"high", N("High")},
	{"lossless", N("Lossless")},
}

// cacheLimits are the sizes the song cache may take, in megabytes.
var cacheLimits = []struct {
	mb    int
	label string
}{
	{0, N("Off")},
	{1024, N("1 GB")},
	{2048, N("2 GB")},
	{5120, N("5 GB")},
	{10240, N("10 GB")},
}
