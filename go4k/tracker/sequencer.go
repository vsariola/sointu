package tracker

import (
	"fmt"
	"sync/atomic"
	"time"
)

func (t *Tracker) TogglePlay() {
	t.Playing = !t.Playing
	t.setPlaying <- t.Playing
}

// sequencerLoop is the main goroutine that handles the playing logic
func (t *Tracker) sequencerLoop(closer chan struct{}) {
	playing := false
	rowTime := (time.Second * 60) / time.Duration(4*t.song.BPM)
	tick := make(<-chan time.Time)
	curVoices := make([]int, len(t.song.Tracks))
	for i := range curVoices {
		curVoices[i] = t.song.FirstTrackVoice(i)
	}
	for {
		select {
		case <-tick:
			next := time.Now().Add(rowTime)
			pattern := atomic.LoadInt32(&t.PlayPattern)
			row := atomic.LoadInt32(&t.PlayRow)
			if int(row+1) == t.song.PatternRows() {
				if int(pattern+1) == t.song.SequenceLength() {
					atomic.StoreInt32(&t.PlayPattern, 0)
				} else {
					atomic.AddInt32(&t.PlayPattern, 1)
				}
				atomic.StoreInt32(&t.PlayRow, 0)
			} else {
				atomic.AddInt32(&t.PlayRow, 1)
			}
			if playing {
				tick = time.After(next.Sub(time.Now()))
			}
			t.playRow(curVoices)
			t.ticked <- struct{}{}
		// TODO: maybe refactor the controls to be nicer, somehow?
		case rowJump := <-t.rowJump:
			atomic.StoreInt32(&t.PlayRow, int32(rowJump))
		case patternJump := <-t.patternJump:
			atomic.StoreInt32(&t.PlayPattern, int32(patternJump))
		case <-closer:
			return
		case playState := <-t.setPlaying:
			playing = playState
			if playing {
				t.playBuffer = make([]float32, t.song.SamplesPerRow())
				tick = time.After(0)
			}
		}
	}
}

// playRow renders and writes the current row
func (t *Tracker) playRow(curVoices []int) {
	pattern := atomic.LoadInt32(&t.PlayPattern)
	row := atomic.LoadInt32(&t.PlayRow)
	for i, trk := range t.song.Tracks {
		patternIndex := trk.Sequence[pattern]
		note := t.song.Patterns[patternIndex][row]
		if note == 1 { // anything but hold causes an action.
			continue // TODO: can hold be actually something else than 1?
		}
		t.synth.Release(curVoices[i])
		if note > 1 {
			curVoices[i]++
			first := t.song.FirstTrackVoice(i)
			if curVoices[i] >= first+trk.NumVoices {
				curVoices[i] = first
			}
			t.synth.Trigger(curVoices[i], note)
		}
	}
	buff := make([]float32, t.song.SamplesPerRow()*2)
	rendered, timeAdvanced, _ := t.synth.Render(buff, t.song.SamplesPerRow())
	err := t.player.Play(buff)
	if err != nil {
		fmt.Println("error playing: %w", err)
	} else if timeAdvanced != t.song.SamplesPerRow() {
		fmt.Println("rendered only", rendered, "/", timeAdvanced, "expected", t.song.SamplesPerRow())
	}
}
