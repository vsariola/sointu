package tracker

import (
	"fmt"
	"math"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	// Player is the audio player for the tracker, run in a separate thread. It
	// is controlled by messages from the model and MIDI messages via the
	// context, typically from the VSTI host. The player sends messages to the
	// model via the playerMessages channel. The model sends messages to the
	// player via the modelMessages channel.
	Player struct {
		synth           sointu.Synth           // the synth used to render audio
		song            sointu.Song            // the song being played
		playing         bool                   // is the player playing the score or not
		rowtime         int                    // how many samples have been played in the current row
		songPos         sointu.SongPos         // the current position in the score
		avgVolumeMeter  VolumeAnalyzer         // the volume analyzer used to calculate the average volume
		peakVolumeMeter VolumeAnalyzer         // the volume analyzer used to calculate the peak volume
		voiceLevels     [vm.MAX_VOICES]float32 // a level that can be used to visualize the volume of each voice
		voices          [vm.MAX_VOICES]voice
		loop            Loop

		recState  recState  // is the recording off; are we waiting for a note; or are we recording
		recording Recording // the recorded MIDI events and BPM

		synther    sointu.Synther // the synther used to create new synths
		playerMsgs chan<- PlayerMsg
		modelMsgs  <-chan interface{}
	}

	// PlayerProcessContext is the context given to the player when processing
	// audio. It is used to get MIDI events and the current BPM.
	PlayerProcessContext interface {
		NextEvent() (event MIDINoteEvent, ok bool)
		BPM() (bpm float64, ok bool)
	}

	// MIDINoteEvent is a MIDI event triggering or releasing a note. In
	// processing, the Frame is relative to the start of the current buffer. In
	// a Recording, the Frame is relative to the start of the recording.
	MIDINoteEvent struct {
		Frame   int
		On      bool
		Channel int
		Note    byte
	}

	// PlayerMsg is a message sent from the player to the model. The Inner
	// field can contain any message. Panic, AverageVolume, PeakVolume, SongRow
	// and VoiceStates transmitted frequently, with every message, so they are
	// treated specially, to avoid boxing. All the rest messages can be boxed to
	// Inner interface{}
	PlayerMsg struct {
		Panic         bool
		AverageVolume Volume
		PeakVolume    Volume
		SongPosition  sointu.SongPos
		VoiceLevels   [vm.MAX_VOICES]float32
		Inner         interface{}
	}
)

type (
	recState int

	voice struct {
		noteID            int
		sustain           bool
		samplesSinceEvent int
	}
)

const (
	recStateNone recState = iota
	recStateWaitingForNote
	recStateRecording
)

const numRenderTries = 10000

// Process renders audio to the given buffer, trying to fill it completely. If
// the buffer is not filled, the synth is destroyed and an error is sent to the
// model. context tells the player which MIDI events happen during the current
// buffer. It is used to trigger and release notes during processing. The
// context is also used to get the current BPM from the host.
func (p *Player) Process(buffer sointu.AudioBuffer, context PlayerProcessContext) {
	p.processMessages(context)
	midi, midiOk := context.NextEvent()
	frame := 0

	if p.recState == recStateRecording {
		p.recording.TotalFrames += len(buffer)
	}

	oldBuffer := buffer

	for i := 0; i < numRenderTries; i++ {
		for midiOk && frame >= midi.Frame {
			if p.recState == recStateWaitingForNote {
				p.recording.TotalFrames = len(buffer)
				p.recState = recStateRecording
			}
			if p.recState == recStateRecording {
				midiTotalFrame := midi
				midiTotalFrame.Frame = p.recording.TotalFrames - len(buffer)
				p.recording.Events = append(p.recording.Events, midiTotalFrame)
			}
			if midi.On {
				p.triggerInstrument(midi.Channel, midi.Note)
			} else {
				p.releaseInstrument(midi.Channel, midi.Note)
			}
			midi, midiOk = context.NextEvent()
		}
		framesUntilMidi := len(buffer)
		if delta := midi.Frame - frame; midiOk && delta < framesUntilMidi {
			framesUntilMidi = delta
		}
		if p.playing && p.rowtime >= p.song.SamplesPerRow() {
			p.advanceRow()
		}
		timeUntilRowAdvance := math.MaxInt32
		if p.playing {
			timeUntilRowAdvance = p.song.SamplesPerRow() - p.rowtime
		}
		if timeUntilRowAdvance < 0 {
			timeUntilRowAdvance = 0
		}
		var rendered, timeAdvanced int
		var err error
		if p.synth != nil {
			rendered, timeAdvanced, err = p.synth.Render(buffer[:framesUntilMidi], timeUntilRowAdvance)
		} else {
			mx := framesUntilMidi
			if timeUntilRowAdvance < mx {
				mx = timeUntilRowAdvance
			}
			for i := 0; i < mx; i++ {
				buffer[i] = [2]float32{}
			}
			rendered = mx
			timeAdvanced = mx
		}
		if err != nil {
			p.synth = nil
			p.send(Alert{Message: fmt.Sprintf("synth.Render: %s", err.Error()), Priority: Error, Name: "PlayerCrash"})
		}
		buffer = buffer[rendered:]
		frame += rendered
		p.rowtime += timeAdvanced
		for i := range p.voices {
			p.voices[i].samplesSinceEvent += rendered
		}
		alpha := float32(math.Exp(-float64(rendered) / 15000))
		for i, state := range p.voices {
			if state.sustain {
				p.voiceLevels[i] = (p.voiceLevels[i]-0.5)*alpha + 0.5
			} else {
				p.voiceLevels[i] *= alpha
			}
		}
		// when the buffer is full, return
		if len(buffer) == 0 {
			err := p.avgVolumeMeter.Update(oldBuffer)
			err2 := p.peakVolumeMeter.Update(oldBuffer)
			if err != nil {
				p.synth = nil
				p.sendAlert("PlayerVolume", err.Error(), Warning)
				return
			}
			if err2 != nil {
				p.synth = nil
				p.sendAlert("PlayerVolume", err2.Error(), Warning)
				return
			}
			p.send(nil)
			return
		}
	}
	// we were not able to fill the buffer with NUM_RENDER_TRIES attempts, destroy synth and throw an error
	p.synth = nil
	p.sendAlert("PlayerCrash", fmt.Sprintf("synth did not fill the audio buffer even with %d render calls", numRenderTries), Error)
}

func (p *Player) advanceRow() {
	if p.song.Score.Length == 0 || p.song.Score.RowsPerPattern == 0 {
		return
	}
	origPos := p.songPos
	p.songPos.PatternRow++ // advance row (this is why we subtracted one in Play())
	if p.loop.Length > 0 && p.songPos.PatternRow >= p.song.Score.RowsPerPattern && p.songPos.OrderRow == p.loop.Start+p.loop.Length-1 {
		p.songPos.PatternRow = 0
		p.songPos.OrderRow = p.loop.Start
	}
	p.songPos = p.song.Score.Clamp(p.songPos)
	if p.songPos == origPos {
		p.send(IsPlayingMsg{bool: false})
		p.playing = false
		for i := range p.song.Score.Tracks {
			p.releaseTrack(i)
		}
		return
	}
	p.send(nil) // just send volume and song row information
	lastVoice := 0
	for i, t := range p.song.Score.Tracks {
		start := lastVoice
		lastVoice = start + t.NumVoices
		n := t.Note(p.songPos)
		switch {
		case n == 0:
			p.releaseTrack(i)
		case n > 1:
			p.triggerTrack(i, n)
		default: // n == 1
		}
	}
	p.rowtime = 0
}

func (p *Player) processMessages(context PlayerProcessContext) {
loop:
	for { // process new message
		select {
		case msg := <-p.modelMsgs:
			switch m := msg.(type) {
			case PanicMsg:
				if m.bool {
					p.synth = nil
				} else {
					p.compileOrUpdateSynth()
				}
			case sointu.Song:
				p.song = m
				p.compileOrUpdateSynth()
			case sointu.Patch:
				p.song.Patch = m
				p.compileOrUpdateSynth()
			case sointu.Score:
				p.song.Score = m
			case Loop:
				p.loop = m
			case IsPlayingMsg:
				p.playing = bool(m.bool)
				if !p.playing {
					for i := range p.song.Score.Tracks {
						p.releaseTrack(i)
					}
				}
			case BPMMsg:
				p.song.BPM = m.int
				p.compileOrUpdateSynth()
			case RowsPerBeatMsg:
				p.song.RowsPerBeat = m.int
				p.compileOrUpdateSynth()
			case StartPlayMsg:
				p.playing = true
				p.songPos = m.SongPos
				p.songPos.PatternRow--
				p.rowtime = math.MaxInt
				for i, t := range p.song.Score.Tracks {
					if !t.Effect {
						// when starting to play from another position, release only non-effect tracks
						p.releaseTrack(i)
					}
				}
			case NoteOnMsg:
				if m.IsInstr {
					p.triggerInstrument(m.Instr, m.Note)
				} else {
					p.triggerTrack(m.Track, m.Note)
				}
			case NoteOffMsg:
				if m.IsInstr {
					p.releaseInstrument(m.Instr, m.Note)
				} else {
					p.releaseTrack(m.Track)
				}
			case RecordingMsg:
				if m.bool {
					p.recState = recStateWaitingForNote
					p.recording = Recording{}
				} else {
					if p.recState == recStateRecording && len(p.recording.Events) > 0 {
						p.recording.BPM, _ = context.BPM()
						p.send(p.recording)
					}
					p.recState = recStateNone
				}
			default:
				// ignore unknown messages
			}
		default:
			break loop
		}
	}
}

func (p *Player) sendAlert(name, message string, priority AlertPriority) {
	p.send(Alert{
		Name:     name,
		Priority: priority,
		Message:  message,
		Duration: defaultAlertDuration,
	})
}

func (p *Player) compileOrUpdateSynth() {
	if p.song.BPM <= 0 {
		return // bpm not set yet
	}
	if p.synth != nil {
		err := p.synth.Update(p.song.Patch, p.song.BPM)
		if err != nil {
			p.synth = nil
			p.sendAlert("PlayerCrash", fmt.Sprintf("synth.Update: %v", err), Error)
			return
		}
	} else {
		var err error
		p.synth, err = p.synther.Synth(p.song.Patch, p.song.BPM)
		if err != nil {
			p.synth = nil
			p.sendAlert("PlayerCrash", fmt.Sprintf("synther.Synth: %v", err), Error)
			return
		}
	}
}

// all sends from player are always non-blocking, to ensure that the player thread cannot end up in a dead-lock
func (p *Player) send(message interface{}) {
	select {
	case p.playerMsgs <- PlayerMsg{Panic: p.synth == nil, AverageVolume: p.avgVolumeMeter.Level, PeakVolume: p.peakVolumeMeter.Level, SongPosition: p.songPos, VoiceLevels: p.voiceLevels, Inner: message}:
	default:
	}
}

func (p *Player) triggerInstrument(instrument int, note byte) {
	ID := idForInstrumentNote(instrument, note)
	p.release(ID)
	if p.song.Patch == nil || instrument < 0 || instrument >= len(p.song.Patch) {
		return
	}
	voiceStart := p.song.Patch.FirstVoiceForInstrument(instrument)
	voiceEnd := voiceStart + p.song.Patch[instrument].NumVoices
	p.trigger(voiceStart, voiceEnd, note, ID)
}

func (p *Player) releaseInstrument(instrument int, note byte) {
	p.release(idForInstrumentNote(instrument, note))
}

func (p *Player) triggerTrack(track int, note byte) {
	ID := idForTrack(track)
	p.release(ID)
	voiceStart := p.song.Score.FirstVoiceForTrack(track)
	voiceEnd := voiceStart + p.song.Score.Tracks[track].NumVoices
	p.trigger(voiceStart, voiceEnd, note, ID)
}

func (p *Player) releaseTrack(track int) {
	p.release(idForTrack(track))
}

func (p *Player) trigger(voiceStart, voiceEnd int, note byte, ID int) {
	if p.synth == nil {
		return
	}
	var age int = 0
	oldestReleased := false
	oldestVoice := 0
	for i := voiceStart; i < voiceEnd; i++ {
		// find a suitable voice to trigger. if the voice has been released,
		// then we prefer to trigger that over a voice that is still playing. in
		// case two voices are both playing or or both are released, we prefer
		// the older one
		if (!p.voices[i].sustain && !oldestReleased) ||
			(!p.voices[i].sustain == oldestReleased && p.voices[i].samplesSinceEvent >= age) {
			oldestVoice = i
			oldestReleased = !p.voices[i].sustain
			age = p.voices[i].samplesSinceEvent
		}
	}
	p.voices[oldestVoice] = voice{noteID: ID, sustain: true, samplesSinceEvent: 0}
	p.voiceLevels[oldestVoice] = 1.0
	if p.synth != nil {
		p.synth.Trigger(oldestVoice, note)
	}
}

func (p *Player) release(ID int) {
	if p.synth == nil {
		return
	}
	for i := range p.voices {
		if p.voices[i].noteID == ID && p.voices[i].sustain {
			p.voices[i].sustain = false
			p.voices[i].samplesSinceEvent = 0
			p.synth.Release(i)
			return
		}
	}
}

// we need to give voices triggered by different sources a identifier who triggered it
// positive values are for voices triggered by instrument jamming i.e. MIDI message from
// host or pressing key on the keyboard
// negative values are for voices triggered by tracks when playing a song
func idForInstrumentNote(instrument int, note byte) int {
	return instrument*256 + int(note)
}

func idForTrack(track int) int {
	return -1 - track
}
