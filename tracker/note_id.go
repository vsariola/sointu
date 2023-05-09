package tracker

// Describes a note triggered either a track or an instrument
// If Go had union or Either types, this would be it, but in absence
// those, this uses a boolean to define if the instrument is defined or the track
type NoteID struct {
	IsInstr bool
	Instr   int
	Track   int
	Note    byte
}

func NoteIDInstr(instr int, note byte) NoteID {
	return NoteID{IsInstr: true, Instr: instr, Note: note}
}

func NoteIDTrack(track int, note byte) NoteID {
	return NoteID{IsInstr: false, Track: track, Note: note}
}
