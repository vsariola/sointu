package tracker

import "github.com/vsariola/sointu"

type Play Model

func (m *Model) Play() *Play { return (*Play)(m) }

// Position returns the current play position as sointu.SongPos.
func (m *Play) Position() sointu.SongPos { return m.playerStatus.SongPos }

// Loop returns the current Loop telling which part of the song is looped.
func (m *Play) Loop() Loop { return m.loop }

// SongRow returns the current order row being played.
func (m *Play) SongRow() int { return m.d.Song.Score.SongRow(m.playerStatus.SongPos) }

// TrackerHidden returns a Bool controlling whether the tracker UI is hidden
// during playback (for example when recording).
func (m *Play) TrackerHidden() Bool { return MakeBoolFromPtr(&m.trackerHidden) }

// FromCurrentPos returns an Action to start playing the song from the current
// cursor position
func (m *Play) FromCurrentPos() Action { return MakeAction((*playCurrentPos)(m)) }

type playCurrentPos Play

func (m *playCurrentPos) Enabled() bool { return !m.trackerHidden }
func (m *playCurrentPos) Do() {
	(*Model)(m).setPanic(false)
	(*Model)(m).setLoop(Loop{})
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{m.d.Cursor.SongPos}))
}

// FromBeginning returns an Action to start playing the song from the beginning.
func (m *Play) FromBeginning() Action { return MakeAction((*playSongStart)(m)) }

type playSongStart Play

func (m *playSongStart) Enabled() bool { return !m.trackerHidden }
func (m *playSongStart) Do() {
	(*Model)(m).setPanic(false)
	(*Model)(m).setLoop(Loop{})
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{}))
}

// FromSelected returns an Action to start playing and looping the currently
// selected patterns.
func (m *Play) FromSelected() Action { return MakeAction((*playSelected)(m)) }

type playSelected Play

func (m *playSelected) Enabled() bool { return !m.trackerHidden }
func (m *playSelected) Do() {
	(*Model)(m).setPanic(false)
	m.playing = true
	l := (*Model)(m).Order().RowList()
	r := l.listRange()
	newLoop := Loop{r.Start, r.End - r.Start}
	(*Model)(m).setLoop(newLoop)
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: r.Start, PatternRow: 0}}))
}

// FromLoopBeginning returns an Action to start playing from the beginning of the
func (m *Play) FromLoopBeginning() Action { return MakeAction((*playFromLoopStart)(m)) }

type playFromLoopStart Play

func (m *playFromLoopStart) Enabled() bool { return !m.trackerHidden }
func (m *playFromLoopStart) Do() {
	(*Model)(m).setPanic(false)
	if m.loop == (Loop{}) {
		(*Play)(m).FromSelected().Do()
		return
	}
	m.playing = true
	TrySend(m.broker.ToPlayer, any(StartPlayMsg{sointu.SongPos{OrderRow: m.loop.Start, PatternRow: 0}}))
}

// Stop returns an Action to stop playing the song.
func (m *Play) Stop() Action { return MakeAction((*stopPlaying)(m)) }

type stopPlaying Play

func (m *stopPlaying) Do() {
	if !m.playing {
		(*Model)(m).setPanic(true)
		(*Model)(m).setLoop(Loop{})
		return
	}
	m.playing = false
	TrySend(m.broker.ToPlayer, any(IsPlayingMsg{false}))
}

// Panicked returns a Bool to toggle whether the synth is in panic mode or not.
func (m *Play) Panicked() Bool { return MakeBool((*playPanicked)(m)) }

type playPanicked Model

func (m *playPanicked) Value() bool       { return m.panic }
func (m *playPanicked) SetValue(val bool) { (*Model)(m).setPanic(val) }

// IsRecording returns a Bool to toggle whether recording is on or off.
func (m *Play) IsRecording() Bool { return MakeBool((*playIsRecording)(m)) }

type playIsRecording Model

func (m *playIsRecording) Value() bool { return (*Model)(m).recording }
func (m *playIsRecording) SetValue(val bool) {
	m.recording = val
	m.trackerHidden = val
	TrySend(m.broker.ToPlayer, any(RecordingMsg{val}))
}

// Started returns a Bool to toggle whether playback has started or not.
func (m *Play) Started() Bool { return MakeBool((*playStarted)(m)) }

type playStarted Play

func (m *playStarted) Value() bool { return m.playing }
func (m *playStarted) SetValue(val bool) {
	m.playing = val
	if m.playing {
		(*Model)(m).setPanic(false)
		TrySend(m.broker.ToPlayer, any(StartPlayMsg{m.d.Cursor.SongPos}))
	} else {
		TrySend(m.broker.ToPlayer, any(IsPlayingMsg{val}))
	}
}
func (m *playStarted) Enabled() bool { return m.playing || !m.trackerHidden }

// IsFollowing returns a Bool to toggle whether user cursors follows the
// playback cursor.
func (m *Play) IsFollowing() Bool { return MakeBoolFromPtr(&m.follow) }

// IsLooping returns a Bool to toggle whether looping is on or off.
func (m *Play) IsLooping() Bool { return MakeBool((*playIsLooping)(m)) }

type playIsLooping Play

func (m *playIsLooping) Value() bool { return m.loop.Length > 0 }
func (t *playIsLooping) SetValue(val bool) {
	m := (*Model)(t)
	newLoop := Loop{}
	if val {
		l := m.Order().RowList()
		r := l.listRange()
		newLoop = Loop{r.Start, r.End - r.Start}
	}
	m.setLoop(newLoop)
}

func (m *Model) setPanic(val bool) {
	if m.panic != val {
		m.panic = val
		TrySend(m.broker.ToPlayer, any(PanicMsg{val}))
	}
}

func (m *Model) setLoop(newLoop Loop) {
	if m.loop != newLoop {
		m.loop = newLoop
		TrySend(m.broker.ToPlayer, any(newLoop))
	}
}

// SyntherIndex returns an Int representing the index of the currently selected
// synther.
func (m *Play) SyntherIndex() Int { return MakeInt((*playSyntherIndex)(m)) }

type playSyntherIndex Play

func (v *playSyntherIndex) Value() int            { return v.syntherIndex }
func (v *playSyntherIndex) Range() RangeInclusive { return RangeInclusive{0, len(v.synthers) - 1} }
func (v *playSyntherIndex) SetValue(value int) bool {
	if value < 0 || value >= len(v.synthers) {
		return false
	}
	v.syntherIndex = value
	TrySend(v.broker.ToPlayer, any(v.synthers[value]))
	return true
}
func (v *playSyntherIndex) StringOf(value int) string {
	if value < 0 || value >= len(v.synthers) {
		return ""
	}
	return v.synthers[value].Name()
}

// SyntherName returns the name of the currently selected synther.
func (v *Play) SyntherName() string { return v.synthers[v.syntherIndex].Name() }

// CPULoad fills the given buffer with CPU load information and returns the
// number of threads filled.
func (m *Play) CPULoad(buf []sointu.CPULoad) int {
	return copy(buf, m.playerStatus.CPULoad[:m.playerStatus.NumThreads])
}
