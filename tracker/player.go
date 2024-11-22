package tracker

import (
	"fmt"
	"math"
	"slices"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	// Player is the audio player for the tracker, run in a separate thread. It
	// is controlled by messages from the model and MIDI messages via the
	// context, typically from the VSTI host. The player sendTargets messages to the
	// model via the playerMessages channel. The model sendTargets messages to the
	// player via the modelMessages channel.
	Player struct {
		synth       sointu.Synth           // the synth used to render audio
		song        sointu.Song            // the song being played
		playing     bool                   // is the player playing the score or not
		rowtime     int                    // how many samples have been played in the current row
		songPos     sointu.SongPos         // the current position in the score
		voiceLevels [vm.MAX_VOICES]float32 // a level that can be used to visualize the volume of each voice
		voices      [vm.MAX_VOICES]voice
		loop        Loop

		recState   recState   // is the recording off; are we waiting for a note; or are we recording
		recording  Recording  // the recorded MIDI events and BPM
		trackInput TrackInput // for events that are played when not recording

		synther sointu.Synther // the synther used to create new synths
		broker  *Broker        // the broker used to communicate with different parts of the tracker
	}

	// PlayerProcessContext is the context given to the player when processing
	// audio. It is used to get MIDI events and the current BPM.
	PlayerProcessContext interface {
		NextEvent(frame int) (event MIDINoteEvent, ok bool)
		FinishBlock(frame int)
		BPM() (bpm float64, ok bool)

		Constraints() PlayerProcessConstraints
	}

	PlayerProcessConstraints struct {
		IsConstrained   bool
		MaxPolyphony    int
		InstrumentIndex int
	}

	// MIDINoteEvent is a MIDI event triggering or releasing a note. In
	// processing, the Frame is relative to the start of the current buffer. In
	// a Recording, the Frame is relative to the start of the recording.
	MIDINoteEvent struct {
		Frame    int
		On       bool
		Channel  int
		Note     byte
		Velocity byte
	}

	// TrackInput is used for the midi-into-track-input, when not recording
	// For now, there is only one Velocity for all Notes. This might evolve.
	TrackInput struct {
		Notes    []byte
		Velocity byte
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

func NewPlayer(broker *Broker, synther sointu.Synther) *Player {
	return &Player{
		broker:  broker,
		synther: synther,
	}
}

// Process renders audio to the given buffer, trying to fill it completely. If
// the buffer is not filled, the synth is destroyed and an error is sent to the
// model. context tells the player which MIDI events happen during the current
// buffer. It is used to trigger and release notes during processing. The
// context is also used to get the current BPM from the host.
func (p *Player) Process(buffer sointu.AudioBuffer, context PlayerProcessContext) {
	p.processMessages(context)
	constraints := context.Constraints()
	_ = constraints

	frame := 0
	midi, midiOk := context.NextEvent(frame)

	if p.recState == recStateRecording {
		p.recording.TotalFrames += len(buffer)
	}

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
			p.handleMidiInput(midi, constraints)
			midi, midiOk = context.NextEvent(frame)
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

		bufPtr := p.broker.GetAudioBuffer() // borrow a buffer from the broker
		*bufPtr = append(*bufPtr, buffer[:rendered]...)
		if len(*bufPtr) == 0 || !trySend(p.broker.ToModel, MsgToModel{Data: bufPtr}) {
			// if the buffer is empty or sending the rendered waveform to Model
			// failed, return the buffer to the broker
			p.broker.PutAudioBuffer(bufPtr)
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
			p.send(nil)
			context.FinishBlock(frame)
			return
		}
	}
	// we were not able to fill the buffer with NUM_RENDER_TRIES attempts, destroy synth and throw an error
	p.synth = nil
	p.SendAlert("PlayerCrash", fmt.Sprintf("synth did not fill the audio buffer even with %d render calls", numRenderTries), Error)
}

func (p *Player) handleMidiInput(midi MIDINoteEvent, constraints PlayerProcessConstraints) {
	instrIndex := midi.Channel
	if constraints.IsConstrained {
		instrIndex = constraints.InstrumentIndex
	}
	if midi.On {
		p.triggerInstrument(instrIndex, midi.Note)
		if p.addTrackInput(midi, constraints) {
			trySend(p.broker.ToModel, MsgToModel{Data: p.trackInput})
		}
	} else {
		p.releaseInstrument(instrIndex, midi.Note)
		p.removeTrackInput(midi)
	}
}

func (p *Player) addTrackInput(midi MIDINoteEvent, c PlayerProcessConstraints) (changed bool) {
	if c.IsConstrained {
		if len(p.trackInput.Notes) == c.MaxPolyphony {
			return false
		} else if len(p.trackInput.Notes) > c.MaxPolyphony {
			p.trackInput.Notes = p.trackInput.Notes[:c.MaxPolyphony]
			return true
		}
	}
	if slices.Contains(p.trackInput.Notes, midi.Note) {
		return false
	}
	p.trackInput.Notes = append(p.trackInput.Notes, midi.Note)
	p.trackInput.Velocity = midi.Velocity
	return true
}

func (p *Player) removeTrackInput(midi MIDINoteEvent) {
	for i, n := range p.trackInput.Notes {
		if n == midi.Note {
			p.trackInput.Notes = append(
				p.trackInput.Notes[:i],
				p.trackInput.Notes[i+1:]...,
			)
		}
	}
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
		case msg := <-p.broker.ToPlayer:
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
				} else {
					trySend(p.broker.ToModel, MsgToModel{Reset: true})
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
				trySend(p.broker.ToModel, MsgToModel{Reset: true})
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

func (p *Player) SendAlert(name, message string, priority AlertPriority) {
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
			p.SendAlert("PlayerCrash", fmt.Sprintf("synth.Update: %v", err), Error)
			return
		}
	} else {
		var err error
		p.synth, err = p.synther.Synth(p.song.Patch, p.song.BPM)
		if err != nil {
			p.synth = nil
			p.SendAlert("PlayerCrash", fmt.Sprintf("synther.Synth: %v", err), Error)
			return
		}
	}
	voice := 0
	for _, instr := range p.song.Patch {
		if instr.Mute {
			for j := 0; j < instr.NumVoices; j++ {
				p.synth.Release(voice + j)
			}
		}
		voice += instr.NumVoices
	}
}

// all sendTargets from player are always non-blocking, to ensure that the player thread cannot end up in a dead-lock
func (p *Player) send(message interface{}) {
	trySend(p.broker.ToModel, MsgToModel{
		HasPanicPosLevels: true,
		Panic:             p.synth == nil,
		SongPosition:      p.songPos,
		VoiceLevels:       p.voiceLevels,
		Data:              message,
	})
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
	instrIndex, err := p.song.Patch.InstrumentForVoice(oldestVoice)
	if err != nil || p.song.Patch[instrIndex].Mute {
		return
	}
	p.voices[oldestVoice] = voice{noteID: ID, sustain: true, samplesSinceEvent: 0}
	p.voiceLevels[oldestVoice] = 1.0
	p.synth.Trigger(oldestVoice, note)
	trySend(p.broker.ToModel, MsgToModel{TriggerChannel: instrIndex + 1})
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
