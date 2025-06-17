package tracker

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

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

		recording Recording // the recorded MIDI events and BPM

		frame       int64         // the current player frame, used to time events
		frameDeltas map[any]int64 // Player.frame (approx.)= event.Timestamp + frameDeltas[event.Source]
		events      NoteEventList

		cpuload float64 // current CPU load of the player, used to adjust the render rate

		synther sointu.Synther // the synther used to create new synths
		broker  *Broker        // the broker used to communicate with different parts of the tracker
	}

	// PlayerProcessContext is the context given to the player when processing
	// audio. Currently it is only used to get BPM from the VSTI host.
	PlayerProcessContext interface {
		BPM() (bpm float64, ok bool)
	}

	// NoteEvent describes triggering or releasing of a note. The timestamps are
	// in frames, and relative to the clock of the event source. Different
	// sources can use different clocks. Player tries to adjust the timestamps
	// so that each note events would fall inside the current processing block,
	// by maintaining an estimate of the delta from the source clock to the
	// player clock.
	NoteEvent struct {
		Timestamp int64 // in frames, relative to whatever clock the source is using
		On        bool
		Channel   int
		Note      byte
		IsTrack   bool // true if "Channel" means track number, false if it means instrument number
		Source    any

		playerTimestamp int64 // the timestamp of the event, adjusted to the player's clock, used to sort events
	}
)

type (
	voice struct {
		triggerEvent      NoteEvent // which event triggered this voice, used to release the voice
		sustain           bool
		samplesSinceEvent int
	}

	NoteEventList []NoteEvent
)

const numRenderTries = 10000

func NewPlayer(broker *Broker, synther sointu.Synther) *Player {
	return &Player{
		broker:      broker,
		synther:     synther,
		frameDeltas: make(map[any]int64),
	}
}

// Process renders audio to the given buffer, trying to fill it completely. If
// the buffer is not filled, the synth is destroyed and an error is sent to the
// model. context tells the player which MIDI events happen during the current
// buffer. It is used to trigger and release notes during processing. The
// context is also used to get the current BPM from the host.
func (p *Player) Process(buffer sointu.AudioBuffer, context PlayerProcessContext) {
	startTime := time.Now()
	startFrame := p.frame

	p.processMessages(context)
	p.events.adjustTimes(p.frameDeltas, p.frame, p.frame+int64(len(buffer)))

	for i := 0; i < numRenderTries; i++ {
		for len(p.events) > 0 && p.events[0].playerTimestamp <= p.frame {
			ev := p.events[0]
			copy(p.events, p.events[1:]) // remove processed events
			p.events = p.events[:len(p.events)-1]
			p.recording.Record(ev, p.frame)
			p.processNoteEvent(ev)
		}
		framesUntilEvent := len(buffer)
		if len(p.events) > 0 {
			framesUntilEvent = min(int(p.events[0].playerTimestamp-p.frame), len(buffer))
		}
		if p.playing && p.rowtime >= p.song.SamplesPerRow() {
			p.advanceRow()
		}
		timeUntilRowAdvance := math.MaxInt32
		if p.playing {
			timeUntilRowAdvance = max(p.song.SamplesPerRow()-p.rowtime, 0)
		}
		var rendered, timeAdvanced int
		var err error
		if p.synth != nil {
			rendered, timeAdvanced, err = p.synth.Render(buffer[:framesUntilEvent], timeUntilRowAdvance)
			if err != nil {
				p.synth = nil
				p.send(Alert{Message: fmt.Sprintf("synth.Render: %s", err.Error()), Priority: Error, Name: "PlayerCrash"})
			}
		} else {
			rendered = min(framesUntilEvent, timeUntilRowAdvance)
			timeAdvanced = rendered
			clear(buffer[:rendered])
		}

		bufPtr := p.broker.GetAudioBuffer() // borrow a buffer from the broker
		*bufPtr = append(*bufPtr, buffer[:rendered]...)
		if len(*bufPtr) == 0 || !TrySend(p.broker.ToModel, MsgToModel{Data: bufPtr}) {
			// if the buffer is empty or sending the rendered waveform to Model
			// failed, return the buffer to the broker
			p.broker.PutAudioBuffer(bufPtr)
		}
		buffer = buffer[rendered:]
		p.frame += int64(rendered)
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
			p.updateCPULoad(time.Since(startTime), p.frame-startFrame)
			p.send(nil)
			return
		}
	}
	// we were not able to fill the buffer with NUM_RENDER_TRIES attempts, destroy synth and throw an error
	p.synth = nil
	p.events = p.events[:0] // clear events, so we don't try to process them again
	p.SendAlert("PlayerCrash", fmt.Sprintf("synth did not fill the audio buffer even with %d render calls", numRenderTries), Error)
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
			p.processNoteEvent(NoteEvent{Channel: i, IsTrack: true, Source: p})
		}
		return
	}
	for i, t := range p.song.Score.Tracks {
		n := t.Note(p.songPos)
		switch {
		case n == 0:
			p.processNoteEvent(NoteEvent{Channel: i, IsTrack: true, Source: p, On: false})
		case n > 1:
			p.processNoteEvent(NoteEvent{Channel: i, IsTrack: true, Source: p, Note: n, On: true})
		} // n = 1 means hold so do nothing
	}
	p.rowtime = 0
	p.send(nil) // just send volume and song row information
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
						p.processNoteEvent(NoteEvent{Channel: i, IsTrack: true, Source: p})
					}
				} else {
					TrySend(p.broker.ToModel, MsgToModel{Reset: true})
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
						p.processNoteEvent(NoteEvent{Channel: i, IsTrack: true, Source: p})
					}
				}
				TrySend(p.broker.ToModel, MsgToModel{Reset: true})
			case NoteEvent:
				p.events = append(p.events, m)
			case RecordingMsg:
				if m.bool {
					p.recording = Recording{State: RecordingWaitingForNote}
				} else {
					if p.recording.State == RecordingStarted && len(p.recording.Events) > 0 {
						p.recording.Finish(p.frame, p.frameDeltas)
						p.recording.BPM, _ = context.BPM()
						p.send(p.recording)
					}
					p.recording = Recording{} // reset recording
				}
			default:
				// ignore unknown messages
			}
		default:
			break loop
		}
	}
}

func (l NoteEventList) adjustTimes(frameDeltas map[any]int64, minFrame, maxFrame int64) {
	// add new sources to the map
	for _, ev := range l {
		if _, ok := frameDeltas[ev.Source]; !ok {
			frameDeltas[ev.Source] = 0 // doesn't matter, we will adjust it immediately after this
		}
	}
	// for each source, calculate the min and max of the frame
	for source, delta := range frameDeltas {
		var srcMinFrame int64 = math.MaxInt64
		var srcMaxFrame int64 = math.MinInt64
		for _, ev := range l {
			if ev.Source != source {
				continue
			}
			if ev.Timestamp < srcMinFrame {
				srcMinFrame = ev.Timestamp
			}
			if ev.Timestamp > srcMaxFrame {
				srcMaxFrame = ev.Timestamp
			}
		}
		if srcMinFrame == math.MaxInt64 || srcMaxFrame == math.MinInt64 {
			continue // no events for this source in this processing block
		}
		// "left" is the difference between the left edge of the source's events
		// and the left edge of the player clock, calculated using the current frameDelta
		left := minFrame - srcMinFrame - delta
		right := maxFrame - srcMaxFrame - delta
		// we try to adjust the frameDelta so that the source's events are
		// within the processing block
		positiveAdjust := min(max(left, 0), max(right, 0)) // always a positive value
		negativeAdjust := max(min(left, 0), min(right, 0)) // always a negative value
		frameDeltas[source] += positiveAdjust + negativeAdjust
	}
	for i, ev := range l {
		l[i].playerTimestamp = ev.Timestamp + frameDeltas[ev.Source]
	}
	// the events should have been sorted already within each source, but they
	// are not necessarily interleaved correctly, so we sort them now
	slices.SortFunc(l, func(a, b NoteEvent) int {
		return cmp.Compare(a.playerTimestamp, b.playerTimestamp)
	})
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
	TrySend(p.broker.ToModel, MsgToModel{HasPanicPosLevels: true, Panic: p.synth == nil, SongPosition: p.songPos, VoiceLevels: p.voiceLevels, CPULoad: p.cpuload, Data: message})
}

func (p *Player) processNoteEvent(ev NoteEvent) {
	if p.synth == nil {
		return
	}
	// release previous voice
	for i := range p.voices {
		if p.voices[i].sustain &&
			p.voices[i].triggerEvent.Source == ev.Source &&
			p.voices[i].triggerEvent.Channel == ev.Channel &&
			p.voices[i].triggerEvent.IsTrack == ev.IsTrack &&
			(ev.IsTrack || (p.voices[i].triggerEvent.Note == ev.Note)) { // tracks don't match the note number when triggering new event, but instrument events do
			p.voices[i].sustain = false
			p.voices[i].samplesSinceEvent = 0
			p.synth.Release(i)
		}
	}
	if !ev.On {
		return
	}
	var voiceStart, voiceEnd int
	if ev.IsTrack {
		if ev.Channel < 0 || ev.Channel >= len(p.song.Score.Tracks) {
			return
		}
		voiceStart = p.song.Score.FirstVoiceForTrack(ev.Channel)
		voiceEnd = voiceStart + p.song.Score.Tracks[ev.Channel].NumVoices
	} else {
		if p.song.Patch == nil || ev.Channel < 0 || ev.Channel >= len(p.song.Patch) {
			return
		}
		voiceStart = p.song.Patch.FirstVoiceForInstrument(ev.Channel)
		voiceEnd = voiceStart + p.song.Patch[ev.Channel].NumVoices
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
	p.voices[oldestVoice] = voice{triggerEvent: ev, sustain: true, samplesSinceEvent: 0}
	p.voiceLevels[oldestVoice] = 1.0
	p.synth.Trigger(oldestVoice, ev.Note)
	TrySend(p.broker.ToModel, MsgToModel{TriggerChannel: instrIndex + 1})
}

func (p *Player) updateCPULoad(duration time.Duration, frames int64) {
	if frames <= 0 {
		return // no frames rendered, so cannot compute CPU load
	}
	realtime := float64(duration) / 1e9
	songtime := float64(frames) / 44100
	newload := realtime / songtime
	alpha := math.Exp(-songtime) // smoothing factor, time constant of 1 second
	p.cpuload = float64(p.cpuload)*alpha + newload*(1-alpha)
}
