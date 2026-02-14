package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/vm"
	"gopkg.in/yaml.v3"
)

// Instrument returns the Instrument view of the model, containing methods to
// manipulate the instruments.
func (m *Model) Instrument() *InstrModel { return (*InstrModel)(m) }

type InstrModel Model

// Add returns an Action to add a new instrument.
func (m *InstrModel) Add() Action { return MakeAction((*addInstrument)(m)) }

type addInstrument InstrModel

func (m *addInstrument) Enabled() bool { return (*Model)(m).d.Song.Patch.NumVoices() < vm.MAX_VOICES }
func (m *addInstrument) Do() {
	defer (*Model)(m).change("AddInstrument", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	p := sointu.Patch{defaultInstrument.Copy()}
	t := []sointu.Track{{NumVoices: 1}}
	_, _, ok := (*Model)(m).addVoices(voiceIndex, p, t, true, (*Model)(m).linkInstrTrack)
	m.changeCancel = !ok
}

// Delete returns an Action to delete the currently selected instrument(s).
func (m *InstrModel) Delete() Action { return MakeAction((*deleteInstrument)(m)) }

type deleteInstrument InstrModel

func (m *deleteInstrument) Enabled() bool { return len((*Model)(m).d.Song.Patch) > 0 }
func (m *deleteInstrument) Do()           { (*Model)(m).Instrument().List().DeleteElements(false) }

// Split returns an Action to split the currently selected instrument, dividing
// the voices as evenly as possible.
func (m *InstrModel) Split() Action { return MakeAction((*splitInstrument)(m)) }

type splitInstrument InstrModel

func (m *splitInstrument) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch) && m.d.Song.Patch[m.d.InstrIndex].NumVoices > 1
}
func (m *splitInstrument) Do() {
	defer (*Model)(m).change("SplitInstrument", SongChange, MajorChange)()
	voiceIndex := m.d.Song.Patch.Copy().FirstVoiceForInstrument(m.d.InstrIndex)
	middle := voiceIndex + (m.d.Song.Patch[m.d.InstrIndex].NumVoices+1)/2
	end := voiceIndex + m.d.Song.Patch[m.d.InstrIndex].NumVoices
	left, ok := VoiceSlice(m.d.Song.Patch, Range{math.MinInt, middle})
	if !ok {
		m.changeCancel = true
		return
	}
	right, ok := VoiceSlice(m.d.Song.Patch, Range{end, math.MaxInt})
	if !ok {
		m.changeCancel = true
		return
	}
	newInstrument := defaultInstrument.Copy()
	(*Model)(m).assignUnitIDs(newInstrument.Units)
	newInstrument.NumVoices = end - middle
	m.d.Song.Patch = append(left, newInstrument)
	m.d.Song.Patch = append(m.d.Song.Patch, right...)
}

// Item returns information about the instrument at a given index.
func (v *InstrModel) Item(i int) (name, title string, maxLevel float32, mute bool, ok bool) {
	if i < 0 || i >= len(v.d.Song.Patch) {
		return "", "", 0, false, false
	}
	name = v.d.Song.Patch[i].Name
	mute = v.d.Song.Patch[i].Mute
	start := v.d.Song.Patch.FirstVoiceForInstrument(i)
	end := start + v.d.Song.Patch[i].NumVoices
	if end >= vm.MAX_VOICES {
		end = vm.MAX_VOICES
	}
	if start < end {
		for _, level := range v.playerStatus.VoiceLevels[start:end] {
			if maxLevel < level {
				maxLevel = level
			}
		}
	}
	if i >= 0 && i < len(v.derived.patch) {
		title = v.derived.patch[i].title
	}
	ok = true
	return
}

// Tab returns an Int representing the currently selected instrument tab.
func (m *InstrModel) Tab() Int { return MakeInt((*instrumentTab)(m)) }

type instrumentTab InstrModel

func (v *instrumentTab) Value() int            { return int(v.d.InstrumentTab) }
func (v *instrumentTab) Range() RangeInclusive { return RangeInclusive{0, int(NumInstrumentTabs) - 1} }
func (v *instrumentTab) SetValue(value int) bool {
	v.d.InstrumentTab = InstrumentTab(value)
	return true
}

// List returns a List of all the instruments in the patch, implementing
// ListData and MutableListData interfaces.
func (m *InstrModel) List() List { return List{(*instrumentList)(m)} }

type instrumentList InstrModel

func (v *instrumentList) Count() int             { return len(v.d.Song.Patch) }
func (v *instrumentList) Selected() int          { return v.d.InstrIndex }
func (v *instrumentList) Selected2() int         { return v.d.InstrIndex2 }
func (v *instrumentList) SetSelected2(value int) { v.d.InstrIndex2 = value }
func (v *instrumentList) SetSelected(value int) {
	v.d.InstrIndex = value
	v.d.UnitIndex = 0
	v.d.UnitIndex2 = 0
	v.d.UnitSearching = false
	v.d.UnitSearchString = ""
}

func (v *instrumentList) Move(r Range, delta int) (ok bool) {
	voiceDelta := 0
	if delta < 0 {
		voiceDelta = -VoiceRange(v.d.Song.Patch, Range{r.Start + delta, r.Start}).Len()
	} else if delta > 0 {
		voiceDelta = VoiceRange(v.d.Song.Patch, Range{r.End, r.End + delta}).Len()
	}
	if voiceDelta == 0 {
		return false
	}
	ranges := MakeMoveRanges(VoiceRange(v.d.Song.Patch, r), voiceDelta)
	return (*Model)(v).sliceInstrumentsTracks(true, v.linkInstrTrack, ranges[:]...)
}

func (v *instrumentList) Delete(r Range) (ok bool) {
	ranges := Complement(VoiceRange(v.d.Song.Patch, r))
	return (*Model)(v).sliceInstrumentsTracks(true, v.linkInstrTrack, ranges[:]...)
}

func (v *instrumentList) Change(n string, severity ChangeSeverity) func() {
	return (*Model)(v).change("Instruments."+n, SongChange, severity)
}

func (v *instrumentList) Cancel() {
	v.changeCancel = true
}

func (v *instrumentList) Marshal(r Range) ([]byte, error) {
	return (*Model)(v).marshalVoices(VoiceRange(v.d.Song.Patch, r))
}

func (m *instrumentList) Unmarshal(data []byte) (r Range, err error) {
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	r, _, ok := (*Model)(m).unmarshalVoices(voiceIndex, data, true, m.linkInstrTrack)
	if !ok {
		return Range{}, fmt.Errorf("unmarshal: unmarshalVoices failed")
	}
	return r, nil
}

// Thread methods
type (
	instrumentThread1 Model
	instrumentThread2 Model
	instrumentThread3 Model
	instrumentThread4 Model
)

func (m *InstrModel) Thread1() Bool            { return MakeBool((*instrumentThread1)(m)) }
func (m *instrumentThread1) Value() bool       { return (*InstrModel)(m).getThreadsBit(0) }
func (m *instrumentThread1) SetValue(val bool) { (*InstrModel)(m).setThreadsBit(0, val) }
func (m *InstrModel) Thread2() Bool            { return MakeBool((*instrumentThread2)(m)) }
func (m *instrumentThread2) Value() bool       { return (*InstrModel)(m).getThreadsBit(1) }
func (m *instrumentThread2) SetValue(val bool) { (*InstrModel)(m).setThreadsBit(1, val) }
func (m *InstrModel) Thread3() Bool            { return MakeBool((*instrumentThread3)(m)) }
func (m *instrumentThread3) Value() bool       { return (*InstrModel)(m).getThreadsBit(2) }
func (m *instrumentThread3) SetValue(val bool) { (*InstrModel)(m).setThreadsBit(2, val) }
func (m *InstrModel) Thread4() Bool            { return MakeBool((*instrumentThread4)(m)) }
func (m *instrumentThread4) Value() bool       { return (*InstrModel)(m).getThreadsBit(3) }
func (m *instrumentThread4) SetValue(val bool) { (*InstrModel)(m).setThreadsBit(3, val) }

func (m *InstrModel) getThreadsBit(bit int) bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	mask := m.d.Song.Patch[m.d.InstrIndex].ThreadMaskM1 + 1
	return mask&(1<<bit) != 0
}

func (m *InstrModel) setThreadsBit(bit int, value bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	mask := m.d.Song.Patch[m.d.InstrIndex].ThreadMaskM1 + 1
	if value {
		mask |= (1 << bit)
	} else {
		mask &^= (1 << bit)
	}
	defer (*Model)(m).change("ThreadBitMask", PatchChange, MinorChange)()
	m.d.Song.Patch[m.d.InstrIndex].ThreadMaskM1 = max(mask-1, -1) // -1 has all threads disabled, we warn about that
	m.warnAboutCrossThreadSends()
	m.warnNoMultithreadSupport()
	m.warnNoThread()
}

func (m *InstrModel) warnAboutCrossThreadSends() {
	for i, instr := range m.d.Song.Patch {
		for _, unit := range instr.Units {
			if unit.Type == "send" {
				targetID, ok := unit.Parameters["target"]
				if !ok {
					continue
				}
				it, _, err := m.d.Song.Patch.FindUnit(targetID)
				if err != nil {
					continue
				}
				if instr.ThreadMaskM1 != m.d.Song.Patch[it].ThreadMaskM1 {
					(*Alerts)(m).AddNamed("CrossThreadSend", fmt.Sprintf("Instrument %d '%s' has a send to instrument %d '%s' but they are not on the same threads, which may cause issues", i+1, instr.Name, it+1, m.d.Song.Patch[it].Name), Warning)
					return
				}
			}
		}
	}
	(*Alerts)(m).ClearNamed("CrossThreadSend")
}

func (m *InstrModel) warnNoMultithreadSupport() {
	for _, instr := range m.d.Song.Patch {
		if instr.ThreadMaskM1 > 0 && !m.curSynther.SupportsMultithreading() {
			(*Alerts)(m).AddNamed("NoMultithreadSupport", "The current synth does not support multithreading and the patch was configured to use more than one thread", Warning)
			return
		}
	}
	(*Alerts)(m).ClearNamed("NoMultithreadSupport")
}

func (m *InstrModel) warnNoThread() {
	for i, instr := range m.d.Song.Patch {
		if instr.ThreadMaskM1 == -1 {
			(*Alerts)(m).AddNamed("NoThread", fmt.Sprintf("Instrument %d '%s' is not rendered on any thread", i+1, instr.Name), Warning)
			return
		}
	}
	(*Alerts)(m).ClearNamed("NoThread")

}

// Mute returns a Bool for muting/unmuting the currently selected instrument(s).
func (m *InstrModel) Mute() Bool { return MakeBool((*muteInstrument)(m)) }

type muteInstrument Model

func (m *muteInstrument) Value() bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	return m.d.Song.Patch[m.d.InstrIndex].Mute
}
func (m *muteInstrument) SetValue(val bool) {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return
	}
	defer (*Model)(m).change("Mute", PatchChange, MinorChange)()
	a, b := min(m.d.InstrIndex, m.d.InstrIndex2), max(m.d.InstrIndex, m.d.InstrIndex2)
	for i := a; i <= b; i++ {
		if i < 0 || i >= len(m.d.Song.Patch) {
			continue
		}
		m.d.Song.Patch[i].Mute = val
	}
}
func (m *muteInstrument) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch)
}

// Solo returns a Bool for soloing/unsoloing the currently selected instrument(s).
func (m *InstrModel) Solo() Bool { return MakeBool((*soloInstrument)(m)) }

type soloInstrument Model

func (m *soloInstrument) Value() bool {
	a, b := min(m.d.InstrIndex, m.d.InstrIndex2), max(m.d.InstrIndex, m.d.InstrIndex2)
	for i := range m.d.Song.Patch {
		if i < 0 || i >= len(m.d.Song.Patch) {
			continue
		}
		if (i >= a && i <= b) == m.d.Song.Patch[i].Mute {
			return false
		}
	}
	return true
}
func (m *soloInstrument) SetValue(val bool) {
	defer (*Model)(m).change("Solo", PatchChange, MinorChange)()
	a, b := min(m.d.InstrIndex, m.d.InstrIndex2), max(m.d.InstrIndex, m.d.InstrIndex2)
	for i := range m.d.Song.Patch {
		if i < 0 || i >= len(m.d.Song.Patch) {
			continue
		}
		m.d.Song.Patch[i].Mute = !(i >= a && i <= b) && val
	}
}
func (m *soloInstrument) Enabled() bool {
	return m.d.InstrIndex >= 0 && m.d.InstrIndex < len(m.d.Song.Patch)
}

// Name returns a String representing the name of the currently selected
// instrument.
func (m *InstrModel) Name() String { return MakeString((*instrumentName)(m)) }

type instrumentName InstrModel

func (v *instrumentName) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Name
}
func (v *instrumentName) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return false
	}
	defer (*Model)(v).change("InstrumentNameString", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Name = value
	return true
}

// Comment returns a String representing the comment of the currently selected
// instrument.
func (m *InstrModel) Comment() String { return MakeString((*instrumentComment)(m)) }

type instrumentComment InstrModel

func (v *instrumentComment) Value() string {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return ""
	}
	return v.d.Song.Patch[v.d.InstrIndex].Comment
}
func (v *instrumentComment) SetValue(value string) bool {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return false
	}
	defer (*Model)(v).change("InstrumentComment", PatchChange, MinorChange)()
	v.d.Song.Patch[v.d.InstrIndex].Comment = value
	return true
}

// Voices returns an Int representing the number of voices for the currently
// selected instrument.
func (m *InstrModel) Voices() Int { return MakeInt((*instrumentVoices)(m)) }

type instrumentVoices InstrModel

func (v *instrumentVoices) Value() int {
	if v.d.InstrIndex < 0 || v.d.InstrIndex >= len(v.d.Song.Patch) {
		return 1
	}
	return max(v.d.Song.Patch[v.d.InstrIndex].NumVoices, 1)
}

func (m *instrumentVoices) SetValue(value int) bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		return false
	}
	defer (*Model)(m).change("InstrumentVoices", SongChange, MinorChange)()
	voiceIndex := m.d.Song.Patch.FirstVoiceForInstrument(m.d.InstrIndex)
	voiceRange := Range{voiceIndex, voiceIndex + m.d.Song.Patch[m.d.InstrIndex].NumVoices}
	ranges := MakeSetLength(voiceRange, value)
	ok := (*Model)(m).sliceInstrumentsTracks(true, m.linkInstrTrack, ranges...)
	if !ok {
		m.changeCancel = true
	}
	return ok
}

func (v *instrumentVoices) Range() RangeInclusive {
	return RangeInclusive{1, (*Model)(v).remainingVoices(true, v.linkInstrTrack) + v.Value()}
}

// Write writes the currently selected instrument to the given io.WriteCloser.
// If the WriteCloser is a file, the file extension is used to determine the
// format (.json for JSON, anything else for YAML).
func (m *InstrModel) Write(w io.WriteCloser) bool {
	if m.d.InstrIndex < 0 || m.d.InstrIndex >= len(m.d.Song.Patch) {
		(*Model)(m).Alerts().Add("No instrument selected", Error)
		return false
	}
	path := ""
	if f, ok := w.(*os.File); ok {
		path = f.Name()
	}
	var extension = filepath.Ext(path)
	var contents []byte
	var err error
	instr := m.d.Song.Patch[m.d.InstrIndex]
	if _, ok := w.(*os.File); ok {
		instr.Name = "" // don't save the instrument name to a file; we'll replace the instruments name with the filename when loading from a file
	}
	if extension == ".json" {
		contents, err = json.Marshal(instr)
	} else {
		contents, err = yaml.Marshal(instr)
	}
	if err != nil {
		(*Model)(m).Alerts().Add(fmt.Sprintf("Error marshaling an instrument file: %v", err), Error)
		return false
	}
	w.Write(contents)
	w.Close()
	return true
}

// Read reads an instrument from the given io.ReadCloser and sets it as the
// currently selected instrument. The format is determined by trying JSON first, then
// YAML, then 4klang Patch, then 4klang Instrument.
func (m *InstrModel) Read(r io.ReadCloser) bool {
	if m.d.InstrIndex < 0 {
		return false
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return false
	}
	r.Close() // if we can't close the file, it's not a big deal, so ignore the error
	var instrument sointu.Instrument
	var errJSON, errYaml, err4ki, err4kp error
	var patch sointu.Patch
	errJSON = json.Unmarshal(b, &instrument)
	if errJSON == nil {
		goto success
	}
	errYaml = yaml.Unmarshal(b, &instrument)
	if errYaml == nil {
		goto success
	}
	patch, err4kp = sointu.Read4klangPatch(bytes.NewReader(b))
	if err4kp == nil {
		defer (*Model)(m).change("LoadInstrument", PatchChange, MajorChange)()
		m.d.Song.Patch = patch
		return true
	}
	instrument, err4ki = sointu.Read4klangInstrument(bytes.NewReader(b))
	if err4ki == nil {
		goto success
	}
	(*Model)(m).Alerts().Add(fmt.Sprintf("Error unmarshaling an instrument file: %v / %v / %v / %v", errYaml, errJSON, err4ki, err4kp), Error)
	return false
success:
	if f, ok := r.(*os.File); ok {
		filename := f.Name()
		// the instrument names are generally junk, replace them with the filename without extension
		instrument.Name = filepath.Base(filename[:len(filename)-len(filepath.Ext(filename))])
	}
	defer (*Model)(m).change("LoadInstrument", PatchChange, MajorChange)()
	for len(m.d.Song.Patch) <= m.d.InstrIndex {
		m.d.Song.Patch = append(m.d.Song.Patch, defaultInstrument.Copy())
	}
	(*Model)(m).assignUnitIDs(instrument.Units)
	m.d.Song.Patch[m.d.InstrIndex].Name = instrument.Name // only copy the relevant fields to preserve the user defined values e.g. NumVoices and MIDI configuration
	m.d.Song.Patch[m.d.InstrIndex].Comment = instrument.Comment
	m.d.Song.Patch[m.d.InstrIndex].Units = instrument.Units
	return true
}
