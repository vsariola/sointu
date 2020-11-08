package go4k_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/vsariola/sointu/go4k"
)

const expectedMarshaled = "{\"BPM\":100,\"Patterns\":[\"QABEACAAAABLAE4AAAAAAA==\"],\"Tracks\":[{\"NumVoices\":1,\"Sequence\":\"AA==\"}],\"SongLength\":0,\"Patch\":[{\"NumVoices\":1,\"Units\":[{\"Type\":\"envelope\",\"Stereo\":false,\"Parameters\":{\"attack\":32,\"decay\":32,\"gain\":128,\"release\":64,\"sustain\":64},\"DelayTimes\":[]},{\"Type\":\"oscillator\",\"Stereo\":false,\"Parameters\":{\"color\":96,\"detune\":64,\"flags\":64,\"gain\":128,\"phase\":0,\"shape\":64,\"transpose\":64},\"DelayTimes\":[]},{\"Type\":\"mulp\",\"Stereo\":false,\"Parameters\":{},\"DelayTimes\":[]},{\"Type\":\"envelope\",\"Stereo\":false,\"Parameters\":{\"attack\":32,\"decay\":32,\"gain\":128,\"release\":64,\"sustain\":64},\"DelayTimes\":[]},{\"Type\":\"oscillator\",\"Stereo\":false,\"Parameters\":{\"color\":64,\"detune\":64,\"flags\":64,\"gain\":128,\"phase\":64,\"shape\":96,\"transpose\":72},\"DelayTimes\":[]},{\"Type\":\"mulp\",\"Stereo\":false,\"Parameters\":{},\"DelayTimes\":[]},{\"Type\":\"out\",\"Stereo\":true,\"Parameters\":{\"gain\":128},\"DelayTimes\":[]}]}]}"

var testSong = go4k.Song{
	BPM:      100,
	Patterns: [][]byte{{64, 0, 68, 0, 32, 0, 0, 0, 75, 0, 78, 0, 0, 0, 0, 0}},
	Tracks: []go4k.Track{
		{NumVoices: 1, Sequence: []byte{0}},
	},
	SongLength: 0,
	Patch: go4k.Patch{
		go4k.Instrument{NumVoices: 1, Units: []go4k.Unit{
			{"envelope", false, map[string]int{"attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}, []int{}},
			{"oscillator", false, map[string]int{"transpose": 64, "detune": 64, "phase": 0, "color": 96, "shape": 64, "gain": 128, "flags": 0x40}, []int{}},
			{"mulp", false, map[string]int{}, []int{}},
			{"envelope", false, map[string]int{"attack": 32, "decay": 32, "sustain": 64, "release": 64, "gain": 128}, []int{}},
			{"oscillator", false, map[string]int{"transpose": 72, "detune": 64, "phase": 64, "color": 64, "shape": 96, "gain": 128, "flags": 0x40}, []int{}},
			{"mulp", false, map[string]int{}, []int{}},
			{"out", true, map[string]int{"gain": 128}, []int{}},
		}},
	},
}

func TestSongMarshalJSON(t *testing.T) {
	songbytes, err := json.Marshal(testSong)
	if err != nil {
		t.Fatalf("cannot marshal song: %v", err)
	}
	if string(songbytes) != expectedMarshaled {
		t.Fatalf("expectedMarshaled song to unexpected result, got %v, expected %v", string(songbytes), expectedMarshaled)
	}
}

func TestSongUnmarshalJSON(t *testing.T) {
	var song go4k.Song
	err := json.Unmarshal([]byte(expectedMarshaled), &song)
	if err != nil {
		t.Fatalf("cannot unmarshal song: %v", err)
	}
	if !reflect.DeepEqual(song, testSong) {
		t.Fatalf("unmarshaled song to unexpected result, got %#v, expected  %#v", song, testSong)
	}
}
