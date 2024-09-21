//go:build plugin

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/vsariola/sointu"
	"github.com/vsariola/sointu/cmd"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gioui"
	"pipelined.dev/audio/vst2"
)

type VSTIProcessContext struct {
	events     []vst2.MIDIEvent
	eventIndex int
	host       vst2.Host
	parameters []*vst2.Parameter
}

func (c *VSTIProcessContext) NextEvent() (event tracker.MIDINoteEvent, ok bool) {
	for c.eventIndex < len(c.events) {
		ev := c.events[c.eventIndex]
		c.eventIndex++
		switch {
		case ev.Data[0] >= 0x80 && ev.Data[0] < 0x90:
			channel := ev.Data[0] - 0x80
			note := ev.Data[1]
			return tracker.MIDINoteEvent{Frame: int(ev.DeltaFrames), On: false, Channel: int(channel), Note: note}, true
		case ev.Data[0] >= 0x90 && ev.Data[0] < 0xA0:
			channel := ev.Data[0] - 0x90
			note := ev.Data[1]
			return tracker.MIDINoteEvent{Frame: int(ev.DeltaFrames), On: true, Channel: int(channel), Note: note}, true
		default:
			// ignore all other MIDI messages
		}
	}
	return tracker.MIDINoteEvent{}, false
}

func (c *VSTIProcessContext) BPM() (bpm float64, ok bool) {
	timeInfo := c.host.GetTimeInfo(vst2.TempoValid)
	if timeInfo == nil || timeInfo.Flags&vst2.TempoValid == 0 || timeInfo.Tempo == 0 {
		return 0, false
	}
	return timeInfo.Tempo, true
}

func (c *VSTIProcessContext) Params() (ret tracker.ExtValueArray, ok bool) {
	for i, p := range c.parameters {
		ret[i] = p.Value
	}
	return ret, true
}

func (c *VSTIProcessContext) SetParams(a tracker.ExtParamArray) bool {
	changed := false
	for i, p := range c.parameters {
		i := i
		name := a[i].Param.Name
		if name == "" {
			name = fmt.Sprintf("P%d", i)
		}
		if p.Value != a[i].Val || p.Name != name {
			p.Value = a[i].Val
			p.Name = name
			p.GetValueFunc = func(value float32) float32 {
				return float32(a[i].Param.MinValue) + value*float32(a[i].Param.MaxValue-a[i].Param.MinValue)
			}
			p.GetValueLabelFunc = func(v float32) string {
				if f := a[i].Param.DisplayFunc; f != nil {
					s, u := a[i].Param.DisplayFunc(int(v + .5))
					c.parameters[i].Unit = u
					return s
				}
				c.parameters[i].Unit = ""
				return strconv.Itoa(int(v + .5))
			}
			changed = true
		}
	}
	if changed {
		c.host.UpdateDisplay()
	}
	return true
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
		model, player := tracker.NewModelPlayer(cmd.MainSynther, recoveryFile)
		t := gioui.NewTracker(model)
		tracker.Bool{BoolData: (*tracker.InstrEnlarged)(model)}.Set(true)
		if s := h.GetSampleRate(); math.Abs(float64(h.GetSampleRate()-44100.0)) > 1e-6 {
			model.Alerts().AddAlert(tracker.Alert{
				Message:  fmt.Sprintf("VSTi host sample rate is %.0f Hz; sointu supports 44100 Hz only", s),
				Priority: tracker.Error,
				Duration: 10 * time.Second,
			})
		}
		go t.Main()
		parameters := make([]*vst2.Parameter, 0, tracker.ExtParamCount)
		for i := 0; i < tracker.ExtParamCount; i++ {
			parameters = append(parameters,
				&vst2.Parameter{
					Name:         fmt.Sprintf("P%d", i),
					NotAutomated: true,
				})
		}
		context := VSTIProcessContext{host: h, parameters: parameters}
		buf := make(sointu.AudioBuffer, 1024)
		return vst2.Plugin{
				UniqueID:       PLUGIN_ID,
				Version:        version,
				InputChannels:  0,
				OutputChannels: 2,
				Name:           PLUGIN_NAME,
				Vendor:         "vsariola/sointu",
				Category:       vst2.PluginCategorySynth,
				Flags:          vst2.PluginIsSynth,
				Parameters:     parameters,
				ProcessFloatFunc: func(in, out vst2.FloatBuffer) {
					left := out.Channel(0)
					right := out.Channel(1)
					if len(buf) < out.Frames {
						buf = append(buf, make(sointu.AudioBuffer, out.Frames-len(buf))...)
					}
					buf = buf[:out.Frames]
					player.Process(buf, &context)
					for i := 0; i < out.Frames; i++ {
						left[i], right[i] = buf[i][0], buf[i][1]
					}
					context.events = context.events[:0] // reset buffer, but keep the allocated memory
					context.eventIndex = 0
				},
			}, vst2.Dispatcher{
				CanDoFunc: func(pcds vst2.PluginCanDoString) vst2.CanDoResponse {
					switch pcds {
					case vst2.PluginCanReceiveEvents, vst2.PluginCanReceiveMIDIEvent, vst2.PluginCanReceiveTimeInfo:
						return vst2.YesCanDo
					}
					return vst2.NoCanDo
				},
				ProcessEventsFunc: func(ev *vst2.EventsPtr) {
					for i := 0; i < ev.NumEvents(); i++ {
						a := ev.Event(i)
						switch v := a.(type) {
						case *vst2.MIDIEvent:
							context.events = append(context.events, *v)
						}
					}
				},
				CloseFunc: func() {
					t.Exec() <- func() { t.ForceQuit().Do() }
					t.WaitQuitted()
				},
				GetChunkFunc: func(isPreset bool) []byte {
					retChn := make(chan []byte)
					t.Exec() <- func() { retChn <- t.MarshalRecovery() }
					return <-retChn
				},
				SetChunkFunc: func(data []byte, isPreset bool) {
					t.Exec() <- func() { t.UnmarshalRecovery(data) }
				},
			}

	}
}

func main() {}
