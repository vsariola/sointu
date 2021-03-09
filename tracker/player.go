package tracker

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/vsariola/sointu"
)

type Player struct {
	packedPos uint64

	playCmds chan uint64

	mutex         sync.Mutex
	runningID     uint32
	voiceNoteID   []uint32
	voiceReleased []bool
	synth         sointu.Synth
	patch         sointu.Patch

	synthNotNil int32
}

type voiceNote struct {
	voice int
	note  byte
}

// Position returns the current play position (song row), and a bool indicating
// if the player is currently playing. The function is threadsafe.
func (p *Player) Position() (SongRow, bool) {
	packedPos := atomic.LoadUint64(&p.packedPos)
	if packedPos == math.MaxUint64 { // stopped
		return SongRow{}, false
	}
	return unpackPosition(packedPos), true
}

func (p *Player) Playing() bool {
	packedPos := atomic.LoadUint64(&p.packedPos)
	if packedPos == math.MaxUint64 { // stopped
		return false
	}
	return true
}

func (p *Player) Play(position SongRow) {
	position.Row-- // we'll advance this very shortly
	p.playCmds <- packPosition(position)
}

func (p *Player) Stop() {
	p.playCmds <- math.MaxUint64
}

func (p *Player) Disable() {
	p.mutex.Lock()
	p.synth = nil
	atomic.StoreInt32(&p.synthNotNil, 0)
	p.mutex.Unlock()
}

func (p *Player) Enabled() bool {
	return atomic.LoadInt32(&p.synthNotNil) == 1
}

func NewPlayer(service sointu.SynthService, closer <-chan struct{}, patchs <-chan sointu.Patch, scores <-chan sointu.Score, samplesPerRows <-chan int, posChanged chan<- struct{}, syncOutput chan<- []float32, outputs ...chan<- []float32) *Player {
	p := &Player{playCmds: make(chan uint64, 16)}
	go func() {
		var score sointu.Score
		buffer := make([]float32, 2048)
		buffer2 := make([]float32, 2048)
		zeros := make([]float32, 2048)
		totalSyncs := 1 // just the beat
		syncBuffer := make([]float32, (2048+255)/256*totalSyncs)
		syncBuffer2 := make([]float32, (2048+255)/256*totalSyncs)
		rowTime := 0
		samplesPerRow := math.MaxInt32
		var trackIDs []uint32
		atomic.StoreUint64(&p.packedPos, math.MaxUint64)
		for {
			select {
			case <-closer:
				for _, o := range outputs {
					close(o)
				}
				return
			case patch := <-patchs:
				p.mutex.Lock()
				p.patch = patch
				if p.synth != nil {
					err := p.synth.Update(patch)
					if err != nil {
						p.synth = nil
						atomic.StoreInt32(&p.synthNotNil, 0)
					}
				} else {
					s, err := service.Compile(patch)
					if err == nil {
						p.synth = s
						atomic.StoreInt32(&p.synthNotNil, 1)
						for i := 0; i < 32; i++ {
							s.Release(i)
						}
					}
				}
				totalSyncs = 1 + p.patch.NumSyncs()
				syncBuffer = make([]float32, ((2048+255)/256)*totalSyncs)
				syncBuffer2 = make([]float32, ((2048+255)/256)*totalSyncs)
				p.mutex.Unlock()
			case score = <-scores:
				if row, playing := p.Position(); playing {
					atomic.StoreUint64(&p.packedPos, packPosition(row.Wrap(score)))
				}
			case samplesPerRow = <-samplesPerRows:
			case packedPos := <-p.playCmds:
				atomic.StoreUint64(&p.packedPos, packedPos)
				if packedPos == math.MaxUint64 {
					p.mutex.Lock()
					for _, id := range trackIDs {
						p.release(id)
					}
					p.mutex.Unlock()
				}
				rowTime = math.MaxInt32
			default:
				row, playing := p.Position()
				if playing && rowTime >= samplesPerRow && score.Length > 0 && score.RowsPerPattern > 0 {
					row.Row++ // advance row (this is why we subtracted one in Play())
					row = row.Wrap(score)
					atomic.StoreUint64(&p.packedPos, packPosition(row))
					select {
					case posChanged <- struct{}{}:
					default:
					}
					p.mutex.Lock()
					lastVoice := 0
					for i, t := range score.Tracks {
						start := lastVoice
						lastVoice = start + t.NumVoices
						if row.Pattern < 0 || row.Pattern >= len(t.Order) {
							continue
						}
						o := t.Order[row.Pattern]
						if o < 0 || o >= len(t.Patterns) {
							continue
						}
						pat := t.Patterns[o]
						if row.Row < 0 || row.Row >= len(pat) {
							continue
						}
						n := pat[row.Row]
						for len(trackIDs) <= i {
							trackIDs = append(trackIDs, 0)
						}
						if n != 1 && trackIDs[i] > 0 {
							p.release(trackIDs[i])
						}
						if n > 1 && p.synth != nil {
							trackIDs[i] = p.trigger(start, lastVoice, n)
						}
					}
					p.mutex.Unlock()
					rowTime = 0
				}
				if p.synth != nil {
					renderTime := samplesPerRow - rowTime
					if !playing {
						renderTime = math.MaxInt32
					}
					p.mutex.Lock()
					rendered, syncs, timeAdvanced, err := p.synth.Render(buffer, syncBuffer, renderTime)
					if err != nil {
						p.synth = nil
						atomic.StoreInt32(&p.synthNotNil, 0)
					}
					p.mutex.Unlock()
					for i := 0; i < syncs; i++ {
						a := syncBuffer[i*totalSyncs]
						b := (a+float32(rowTime))/float32(samplesPerRow) + float32(row.Pattern*score.RowsPerPattern+row.Row)
						syncBuffer[i*totalSyncs] = b
					}
					rowTime += timeAdvanced
					for window := syncBuffer[:totalSyncs*syncs]; len(window) > 0; window = window[totalSyncs:] {
						select {
						case syncOutput <- window[:totalSyncs]:
						default:
						}
					}
					for _, o := range outputs {
						o <- buffer[:rendered*2]
					}
					buffer2, buffer = buffer, buffer2
					syncBuffer2, syncBuffer = syncBuffer, syncBuffer2
				} else {
					rowTime += len(zeros) / 2
					for _, o := range outputs {
						o <- zeros
					}
				}
			}
		}
	}()
	return p
}

// Trigger is used to manually play a note on the sequencer when jamming. It is
// thread-safe. It starts to play one of the voice in the range voiceStart
// (inclusive) and voiceEnd (exclusive). It returns a id that can be called to
// release the voice playing the note (in case the voice has not been captured
// by someone else already).
func (p *Player) Trigger(voiceStart, voiceEnd int, note byte) uint32 {
	if note <= 1 {
		return 0
	}
	p.mutex.Lock()
	id := p.trigger(voiceStart, voiceEnd, note)
	p.mutex.Unlock()
	return id
}

// Release is used to manually release a note on the player when jamming.
// Expects an ID that was previously acquired by calling Trigger.
func (p *Player) Release(ID uint32) {
	if ID == 0 {
		return
	}
	p.mutex.Lock()
	p.release(ID)
	p.mutex.Unlock()
}

func (p *Player) trigger(voiceStart, voiceEnd int, note byte) uint32 {
	if p.synth == nil {
		return 0
	}
	var oldestID uint32 = math.MaxUint32
	p.runningID++
	newID := p.runningID
	oldestReleased := false
	oldestVoice := 0
	for i := voiceStart; i < voiceEnd; i++ {
		for len(p.voiceReleased) <= i {
			p.voiceReleased = append(p.voiceReleased, true)
		}
		for len(p.voiceNoteID) <= i {
			p.voiceNoteID = append(p.voiceNoteID, 0)
		}
		// find a suitable voice to trigger. if the voice has been released,
		// then we prefer to trigger that over a voice that is still playing. in
		// case two voices are both playing or or both are released, we prefer
		// the older one
		id := p.voiceNoteID[i]
		isReleased := p.voiceReleased[i]
		if id < oldestID && (oldestReleased == isReleased) || (!oldestReleased && isReleased) {
			oldestVoice = i
			oldestID = id
			oldestReleased = isReleased
		}
	}
	p.voiceNoteID[oldestVoice] = newID
	p.voiceReleased[oldestVoice] = false
	if p.synth != nil {
		p.synth.Trigger(oldestVoice, note)
	}
	return newID
}

func (p *Player) release(ID uint32) {
	if p.synth == nil {
		return
	}
	for i := 0; i < len(p.voiceNoteID); i++ {
		if p.voiceNoteID[i] == ID && !p.voiceReleased[i] {
			p.voiceReleased[i] = true
			p.synth.Release(i)
			return
		}
	}
}

func packPosition(pos SongRow) uint64 {
	return (uint64(uint32(pos.Pattern)) << 32) + uint64(uint32(pos.Row))
}

func unpackPosition(packedPos uint64) SongRow {
	pattern := int(int32(packedPos >> 32))
	row := int(int32(packedPos & 0xFFFFFFFF))
	return SongRow{Pattern: pattern, Row: row}
}
