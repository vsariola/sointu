package tracker

import (
	"fmt"
	"math"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
)

type (
	Player struct {
		voiceNoteID       []int
		voiceReleased     []bool
		synth             sointu.Synth
		patch             sointu.Patch
		score             sointu.Score
		playing           bool
		rowtime           int
		position          SongRow
		samplesSinceEvent []int
		samplesPerRow     int
		bpm               int
		volume            Volume
		voiceStates       [vm.MAX_VOICES]float32

		recording            bool
		recordingNoteArrived bool
		recordingFrames      int
		recordingEvents      []PlayerProcessEvent

		synthService   sointu.SynthService
		playerMessages chan<- PlayerMessage
		modelMessages  <-chan interface{}
	}

	PlayerProcessContext interface {
		NextEvent() (event PlayerProcessEvent, ok bool)
		BPM() (bpm float64, ok bool)
	}

	PlayerProcessEvent struct {
		Frame   int
		On      bool
		Channel int
		Note    byte
	}

	PlayerPlayingMessage struct {
		bool
	}

	PlayerRecordedMessage struct {
		BPM         float64 // vsts allow bpms as floats so for accurate reconstruction, keep it as float for recording
		Events      []PlayerProcessEvent
		TotalFrames int
	}

	// Volume and SongRow are transmitted so frequently that they are treated specially, to avoid boxing. All the
	// rest messages can be boxed to interface{}
	PlayerMessage struct {
		Volume      Volume
		SongRow     SongRow
		VoiceStates [vm.MAX_VOICES]float32
		Inner       interface{}
	}

	PlayerCrashMessage struct {
		error
	}

	PlayerVolumeErrorMessage struct {
		error
	}

	voiceNote struct {
		voice int
		note  byte
	}

	recordEvent struct {
		frame int
	}
)

const NUM_RENDER_TRIES = 10000

func NewPlayer(synthService sointu.SynthService, playerMessages chan<- PlayerMessage, modelMessages <-chan interface{}) *Player {
	p := &Player{
		playerMessages: playerMessages,
		modelMessages:  modelMessages,
		synthService:   synthService,
		volume:         Volume{Average: [2]float64{1e-9, 1e-9}, Peak: [2]float64{1e-9, 1e-9}},
	}
	return p
}

func (p *Player) Process(buffer []float32, context PlayerProcessContext) {
	p.processMessages(context)
	midi, midiOk := context.NextEvent()
	frame := 0

	if p.recording && p.recordingNoteArrived {
		p.recordingFrames += len(buffer) / 2
	}

	oldBuffer := buffer

	for i := 0; i < NUM_RENDER_TRIES; i++ {
		for midiOk && frame >= midi.Frame {
			if p.recording {
				if !p.recordingNoteArrived {
					p.recordingFrames = len(buffer) / 2
					p.recordingNoteArrived = true
				}
				midiTotalFrame := midi
				midiTotalFrame.Frame = p.recordingFrames - len(buffer)/2
				p.recordingEvents = append(p.recordingEvents, midiTotalFrame)
			}
			if midi.On {
				p.triggerInstrument(midi.Channel, midi.Note)
			} else {
				p.releaseInstrument(midi.Channel, midi.Note)
			}
			midi, midiOk = context.NextEvent()
		}
		framesUntilMidi := len(buffer) / 2
		if delta := midi.Frame - frame; midiOk && delta < framesUntilMidi {
			framesUntilMidi = delta
		}
		if p.playing && p.rowtime >= p.samplesPerRow {
			p.advanceRow()
		}
		timeUntilRowAdvance := math.MaxInt32
		if p.playing {
			timeUntilRowAdvance = p.samplesPerRow - p.rowtime
		}
		var rendered, timeAdvanced int
		var err error
		if p.synth != nil {
			rendered, timeAdvanced, err = p.synth.Render(buffer[:framesUntilMidi*2], timeUntilRowAdvance)
		} else {
			mx := framesUntilMidi
			if timeUntilRowAdvance < mx {
				mx = timeUntilRowAdvance
			}
			for i := 0; i < mx*2; i++ {
				buffer[i] = 0
			}
			rendered = mx
			timeAdvanced = mx
		}
		if err != nil {
			p.synth = nil
			p.trySend(PlayerCrashMessage{fmt.Errorf("synth.Render: %w", err)})
		}
		buffer = buffer[rendered*2:]
		frame += rendered
		p.rowtime += timeAdvanced
		for i := range p.samplesSinceEvent {
			p.samplesSinceEvent[i] += rendered
		}
		alpha := float32(math.Exp(-float64(rendered) / 15000))
		for i, released := range p.voiceReleased {
			if released {
				p.voiceStates[i] *= alpha
			} else {
				p.voiceStates[i] = (p.voiceStates[i]-0.5)*alpha + 0.5
			}
		}
		// when the buffer is full, return
		if len(buffer) == 0 {
			err := p.volume.Analyze(oldBuffer, 0.3, 1e-4, 1, -100, 20)
			var msg interface{}
			if err != nil {
				msg = PlayerVolumeErrorMessage{err}
			}
			p.trySend(msg)
			return
		}
	}
	// we were not able to fill the buffer with NUM_RENDER_TRIES attempts, destroy synth and throw an error
	p.synth = nil
	p.trySend(PlayerCrashMessage{fmt.Errorf("synth did not fill the audio buffer even with %d render calls", NUM_RENDER_TRIES)})
}

func (p *Player) advanceRow() {
	if p.score.Length == 0 || p.score.RowsPerPattern == 0 {
		return
	}
	p.position.Row++ // advance row (this is why we subtracted one in Play())
	p.position = p.position.Wrap(p.score)
	p.trySend(nil) // just send volume and song row information
	lastVoice := 0
	for i, t := range p.score.Tracks {
		start := lastVoice
		lastVoice = start + t.NumVoices
		if p.position.Pattern < 0 || p.position.Pattern >= len(t.Order) {
			continue
		}
		o := t.Order[p.position.Pattern]
		if o < 0 || o >= len(t.Patterns) {
			continue
		}
		pat := t.Patterns[o]
		if p.position.Row < 0 || p.position.Row >= len(pat) {
			continue
		}
		n := pat[p.position.Row]
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
		case msg := <-p.modelMessages:
			switch m := msg.(type) {
			case ModelPanicMessage:
				if m.bool {
					p.synth = nil
				} else {
					p.compileOrUpdateSynth()
				}
			case ModelPatchChangedMessage:
				p.patch = m.Patch
				p.compileOrUpdateSynth()
			case ModelScoreChangedMessage:
				p.score = m.Score
			case ModelPlayingChangedMessage:
				p.playing = m.bool
				if !p.playing {
					for i := range p.score.Tracks {
						p.releaseTrack(i)
					}
				}
			case ModelSamplesPerRowChangedMessage:
				p.samplesPerRow = 44100 * 60 / (m.BPM * m.RowsPerBeat)
				p.bpm = m.BPM
				p.compileOrUpdateSynth()
			case ModelPlayFromPositionMessage:
				p.playing = true
				p.position = m.SongRow
				p.position.Row--
				p.rowtime = math.MaxInt
				for i, t := range p.score.Tracks {
					if !t.Effect {
						// when starting to play from another position, release only non-effect tracks
						p.releaseTrack(i)
					}
				}
			case ModelNoteOnMessage:
				if m.id.IsInstr {
					p.triggerInstrument(m.id.Instr, m.id.Note)
				} else {
					p.triggerTrack(m.id.Track, m.id.Note)
				}
			case ModelNoteOffMessage:
				if m.id.IsInstr {
					p.releaseInstrument(m.id.Instr, m.id.Note)
				} else {
					p.releaseTrack(m.id.Track)
				}
			case ModelRecordingMessage:
				if m.bool {
					p.recording = true
					p.recordingEvents = make([]PlayerProcessEvent, 0)
					p.recordingFrames = 0
					p.recordingNoteArrived = false
				} else {
					if p.recording && len(p.recordingEvents) > 0 {
						bpm, _ := context.BPM()
						p.trySend(PlayerRecordedMessage{
							BPM:         bpm,
							Events:      p.recordingEvents,
							TotalFrames: p.recordingFrames,
						})
					}
					p.recording = false
				}
			default:
				// ignore unknown messages
			}
		default:
			break loop
		}
	}
}

func (p *Player) compileOrUpdateSynth() {
	if p.synth != nil {
		err := p.synth.Update(p.patch, p.bpm)
		if err != nil {
			p.synth = nil
			p.trySend(PlayerCrashMessage{fmt.Errorf("synth.Update: %w", err)})
			return
		}
	} else {
		var err error
		p.synth, err = p.synthService.Compile(p.patch, p.bpm)
		if err != nil {
			p.synth = nil
			p.trySend(PlayerCrashMessage{fmt.Errorf("synthService.Compile: %w", err)})
			return
		}
		for i := 0; i < 32; i++ {
			p.synth.Release(i)
		}
	}
}

// all sends from player are always non-blocking, to ensure that the player thread cannot end up in a dead-lock
func (p *Player) trySend(message interface{}) {
	select {
	case p.playerMessages <- PlayerMessage{Volume: p.volume, SongRow: p.position, VoiceStates: p.voiceStates, Inner: message}:
	default:
	}
}

func (p *Player) triggerInstrument(instrument int, note byte) {
	ID := idForInstrumentNote(instrument, note)
	p.release(ID)
	voiceStart := p.patch.FirstVoiceForInstrument(instrument)
	voiceEnd := voiceStart + p.patch[instrument].NumVoices
	p.trigger(voiceStart, voiceEnd, note, ID)
}

func (p *Player) releaseInstrument(instrument int, note byte) {
	p.release(idForInstrumentNote(instrument, note))
}

func (p *Player) triggerTrack(track int, note byte) {
	ID := idForTrack(track)
	p.release(ID)
	voiceStart := p.score.FirstVoiceForTrack(track)
	voiceEnd := voiceStart + p.score.Tracks[track].NumVoices
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
		for len(p.voiceReleased) <= i {
			p.voiceReleased = append(p.voiceReleased, true)
		}
		for len(p.samplesSinceEvent) <= i {
			p.samplesSinceEvent = append(p.samplesSinceEvent, 0)
		}
		for len(p.voiceNoteID) <= i {
			p.voiceNoteID = append(p.voiceNoteID, 0)
		}
		// find a suitable voice to trigger. if the voice has been released,
		// then we prefer to trigger that over a voice that is still playing. in
		// case two voices are both playing or or both are released, we prefer
		// the older one
		if (p.voiceReleased[i] && !oldestReleased) ||
			(p.voiceReleased[i] == oldestReleased && p.samplesSinceEvent[i] >= age) {
			oldestVoice = i
			oldestReleased = p.voiceReleased[i]
			age = p.samplesSinceEvent[i]
		}
	}
	p.voiceNoteID[oldestVoice] = ID
	p.voiceReleased[oldestVoice] = false
	p.voiceStates[oldestVoice] = 1.0
	p.samplesSinceEvent[oldestVoice] = 0
	if p.synth != nil {
		p.synth.Trigger(oldestVoice, note)
	}
}

func (p *Player) release(ID int) {
	if p.synth == nil {
		return
	}
	for i := 0; i < len(p.voiceNoteID); i++ {
		if p.voiceNoteID[i] == ID && !p.voiceReleased[i] {
			p.voiceReleased[i] = true
			p.samplesSinceEvent[i] = 0
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
