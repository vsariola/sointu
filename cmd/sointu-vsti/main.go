//go:build plugin

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/cmd"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gioui"
	"pipelined.dev/audio/vst2"
)

type (
	VSTIProcessContext struct {
		events     []vst2.MIDIEvent
		eventIndex int
		host       vst2.Host
	}
)

func (c *VSTIProcessContext) BPM() (bpm float64, ok bool) {
	timeInfo := c.host.GetTimeInfo(vst2.TempoValid)
	if timeInfo == nil || timeInfo.Flags&vst2.TempoValid == 0 || timeInfo.Tempo == 0 {
		return 0, false
	}
	return timeInfo.Tempo, true
}

func init() {
	var (
		version = int32(100)
	)
	vst2.PluginAllocator = func(h vst2.Host) (vst2.Plugin, vst2.Dispatcher) {
		recoveryFile := ""
		if configDir, err := os.UserConfigDir(); err == nil {
			randBytes := make([]byte, 16)
			rand.Read(randBytes)
			recoveryFile = filepath.Join(configDir, "sointu", "sointu-vsti-recovery-"+hex.EncodeToString(randBytes))
		}
		broker := tracker.NewBroker()
		model := tracker.NewModel(broker, cmd.Synthers, cmd.NewMidiContext(broker), recoveryFile)
		player := tracker.NewPlayer(broker, cmd.Synthers[0])
		detector := tracker.NewDetector(broker)
		go detector.Run()

		t := gioui.NewTracker(model)
		model.InstrEnlarged().SetValue(true)
		// since the VST is usually working without any regard for the tracks
		// until recording, disable the Instrument-Track linking by default
		// because it might just confuse the user why instrument cannot be
		// swapped/added etc.
		model.LinkInstrTrack().SetValue(false)
		go t.Main()
		context := &VSTIProcessContext{host: h}
		buf := make(sointu.AudioBuffer, 1024)
		var totalFrames int64 = 0
		return vst2.Plugin{
				UniqueID:       [4]byte{'S', 'n', 't', 'u'},
				Version:        version,
				InputChannels:  0,
				OutputChannels: 2,
				Name:           "Sointu",
				Vendor:         "vsariola/sointu",
				Category:       vst2.PluginCategorySynth,
				Flags:          vst2.PluginIsSynth,
				ProcessFloatFunc: func(in, out vst2.FloatBuffer) {
					if s := h.GetSampleRate(); math.Abs(float64(h.GetSampleRate()-44100.0)) > 1e-6 {
						player.SendAlert("WrongSampleRate", fmt.Sprintf("VSTi host sample rate is %.0f Hz; sointu supports 44100 Hz only", s), tracker.Error)
					}
					left := out.Channel(0)
					right := out.Channel(1)
					if len(buf) < out.Frames {
						buf = append(buf, make(sointu.AudioBuffer, out.Frames-len(buf))...)
					}
					buf = buf[:out.Frames]
					player.Process(buf, context)
					for i := 0; i < out.Frames; i++ {
						left[i], right[i] = buf[i][0], buf[i][1]
					}
					totalFrames += int64(out.Frames)
				},
			}, vst2.Dispatcher{
				CanDoFunc: func(pcds vst2.PluginCanDoString) vst2.CanDoResponse {
					switch pcds {
					case vst2.PluginCanReceiveEvents, vst2.PluginCanReceiveMIDIEvent, vst2.PluginCanReceiveTimeInfo:
						return vst2.YesCanDo
					}
					return vst2.NoCanDo
				},
				ProcessEventsFunc: func(events *vst2.EventsPtr) {
					for i := 0; i < events.NumEvents(); i++ {
						switch ev := events.Event(i).(type) {
						case *vst2.MIDIEvent:
							if ev.Data[0] >= 0x80 && ev.Data[0] <= 0x9F {
								channel := ev.Data[0] & 0x0F
								note := ev.Data[1]
								on := ev.Data[0] >= 0x90
								trackerEvent := tracker.NoteEvent{Timestamp: int64(ev.DeltaFrames) + totalFrames, On: on, Channel: int(channel), Note: note, Source: &context}
								tracker.TrySend(broker.MIDIChannel(), any(trackerEvent))
							}
						}
					}
				},
				CloseFunc: func() {
					tracker.TrySend(broker.CloseDetector, struct{}{})
					tracker.TrySend(broker.CloseGUI, struct{}{})
					tracker.TimeoutReceive(broker.FinishedDetector, 3*time.Second)
					tracker.TimeoutReceive(broker.FinishedGUI, 3*time.Second)
				},
				GetChunkFunc: func(isPreset bool) []byte {
					retChn := make(chan []byte)

					if !tracker.TrySend(broker.ToModel, tracker.MsgToModel{Data: func() { retChn <- t.MarshalRecovery() }}) {
						return nil
					}
					ret, _ := tracker.TimeoutReceive(retChn, 5*time.Second) // ret will be nil if timeout or channel closed
					return ret
				},
				SetChunkFunc: func(data []byte, isPreset bool) {
					tracker.TrySend(broker.ToModel, tracker.MsgToModel{Data: func() { t.UnmarshalRecovery(data) }})
				},
			}

	}
}

func main() {}
