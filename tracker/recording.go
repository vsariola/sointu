package tracker

import (
	"bytes"
	"errors"
	"math"

	"github.com/vsariola/sointu"
)

type (
	Recording struct {
		BPM                  float64 // vsts allow bpms as floats so for accurate reconstruction, keep it as float for recording
		Events               NoteEventList
		StartFrame, EndFrame int64
		State                RecordingState
	}

	RecordingState int
)

const (
	RecordingNone RecordingState = iota
	RecordingWaitingForNote
	RecordingStarted  // StartFrame is set, but EndFrame is not
	RecordingFinished // StartFrame and EndFrame are both set, recording is finished
)

var ErrInvalidRows = errors.New("rows per beat and rows per pattern must be greater than 1")
var ErrNotFinished = errors.New("the recording was not finished")

func (r *Recording) Record(ev NoteEvent, frame int64) {
	if r.State == RecordingNone || r.State == RecordingFinished {
		return
	}
	if r.State == RecordingWaitingForNote {
		r.StartFrame = frame
		r.State = RecordingStarted
	}
	r.Events = append(r.Events, ev)
}

func (r *Recording) Finish(frame int64, frameDeltas map[any]int64) {
	if r.State != RecordingStarted {
		return
	}
	r.State = RecordingFinished
	r.EndFrame = frame
	r.Events.adjustTimes(frameDeltas, r.StartFrame, r.EndFrame)
}

func (recording *Recording) Score(patch sointu.Patch, rowsPerBeat, rowsPerPattern int) (sointu.Score, error) {
	if rowsPerBeat <= 1 || rowsPerPattern <= 1 {
		return sointu.Score{}, ErrInvalidRows
	}
	if recording.State != RecordingFinished {
		return sointu.Score{}, ErrNotFinished
	}
	type recordingNote struct {
		note     byte
		startRow int
		endRow   int
	}
	channelNotes := make([][]recordingNote, 0)
	// find the length of each note and assign it to its respective channel
	for i, m := range recording.Events {
		if !m.On || m.Channel >= len(patch) || m.IsTrack {
			continue
		}
		var endFrame int64 = math.MaxInt64
		for j := i + 1; j < len(recording.Events); j++ {
			if recording.Events[j].Channel == m.Channel && recording.Events[j].Note == m.Note {
				endFrame = recording.Events[j].playerTimestamp
				break
			}
		}
		for len(channelNotes) <= m.Channel {
			channelNotes = append(channelNotes, make([]recordingNote, 0))
		}
		startRow := frameToRow(recording.BPM, rowsPerBeat, m.playerTimestamp-recording.StartFrame)
		endRow := frameToRow(recording.BPM, rowsPerBeat, endFrame-recording.StartFrame)
		channelNotes[m.Channel] = append(channelNotes[m.Channel], recordingNote{m.Note, startRow, endRow})
	}
	//assign notes to tracks, assigning it to left most track that is released
	//   if none is released, assign it to new track if there's any. otherwise, assign it to the left most track
	tracks := make([][][]recordingNote, len(channelNotes))
	for i, c := range channelNotes {
		tracks[i] = make([][]recordingNote, 0)
	noteloop:
		for _, n := range c {
			// if a track is release, assign the note to left-most released track
			for k, t := range tracks[i] {
				if len(t) == 0 || t[len(t)-1].endRow + patch[i].VoiceRecordingCooldown <= n.startRow {
					tracks[i][k] = append(t, n)
					continue noteloop
				}
			}
			// if there's space for more tracks, create one
			if len(tracks[i]) < patch[i].NumVoices {
				tracks[i] = append(tracks[i], []recordingNote{n})
				continue noteloop
			}
			// otherwise, put the note to the track that was triggered longest time ago
			oldestIndex := -1
			oldestRow := math.MaxInt
			for k, t := range tracks[i] {
				if r := t[len(t)-1].startRow; r < oldestRow {
					oldestRow = r
					oldestIndex = k
				}
			}
			tracks[i][oldestIndex] = append(tracks[i][oldestIndex], n)
		}
	}
	// if there was tracks that had no notes, create empty tracks for them
	for i := range channelNotes {
		if l := len(tracks[i]); l == 0 && l < patch[i].NumVoices {
			tracks[i] = append(tracks[i], []recordingNote{})
		}
	}
	songLengthPatterns := (frameToRow(recording.BPM, rowsPerBeat, recording.EndFrame-recording.StartFrame) + rowsPerPattern - 1) / rowsPerPattern
	songLengthRows := songLengthPatterns * rowsPerPattern
	songTracks := make([]sointu.Track, 0)
	for i, tg := range tracks {
		for j, t := range tg {
			// construct flat linear note arrays for tracks
			flatPattern := make(sointu.Pattern, songLengthRows)
			for k := range flatPattern {
				flatPattern[k] = 1 // set all notes as holds at first
			}
			for _, n := range t {
				if n.startRow >= songLengthRows {
					continue
				}
				flatPattern[n.startRow] = n.note
				if n.endRow < songLengthRows {
					for l := n.startRow + 1; l < n.endRow; l++ {
						flatPattern[l] = 1
					}
					flatPattern[n.endRow] = 0
				} else {
					for l := n.startRow + 1; l < songLengthRows; l++ {
						flatPattern[l] = 1
					}
				}
			}
			// calculate number of voices, distributing the total number of voices to the different tracks
			numVoices := (patch[i].NumVoices + len(tg) - j - 1) / len(tg)
			// construct patterns
			order := make(sointu.Order, songLengthPatterns)
			patterns := make([]sointu.Pattern, 0)
		L:
			for k := range order {
				p := flatPattern[k*rowsPerPattern : (k+1)*rowsPerPattern]
				allHolds := true
				for _, n := range p {
					if n != 1 {
						allHolds = false
						break
					}
				}
				if allHolds {
					order[k] = -1
					continue L
				}
				for l, p2 := range patterns {
					if bytes.Equal(p, p2) {
						order[k] = l
						continue L
					}
				}
				// make a copy of the slice so they are all independent and don't accidentally expand to same memory
				newPat := make(sointu.Pattern, len(p))
				copy(newPat, p)
				order[k] = len(patterns)
				patterns = append(patterns, newPat)
			}
			track := sointu.Track{NumVoices: numVoices, Effect: patch[i].MIDI.Velocity, Order: order, Patterns: patterns}
			songTracks = append(songTracks, track)
		}
	}
	score := sointu.Score{Length: songLengthPatterns, RowsPerPattern: rowsPerPattern, Tracks: songTracks}
	return score, nil
}

func frameToRow(BPM float64, rowsPerBeat int, frame int64) int {
	return int(float64(frame)/44100/60*BPM*float64(rowsPerBeat) + 0.5)
}
