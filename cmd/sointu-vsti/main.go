//go:build plugin

package main

import (
	"github.com/vsariola/sointu/cmd"
	"github.com/vsariola/sointu/tracker"
	"github.com/vsariola/sointu/tracker/gioui"
	"pipelined.dev/audio/vst2"
)

type VSTIProcessContext struct {
	events []vst2.MIDIEvent
	host   vst2.Host
}

func (c *VSTIProcessContext) NextEvent() (event tracker.PlayerProcessEvent, ok bool) {
	var ev vst2.MIDIEvent
	for len(c.events) > 0 {
		ev, c.events = c.events[0], c.events[1:]
		switch {
		case ev.Data[0] >= 0x80 && ev.Data[0] < 0x90:
			channel := ev.Data[0] - 0x80
			note := ev.Data[1]
			return tracker.PlayerProcessEvent{Frame: int(ev.DeltaFrames), On: false, Channel: int(channel), Note: note}, true
		case ev.Data[0] >= 0x90 && ev.Data[0] < 0xA0:
			channel := ev.Data[0] - 0x90
			note := ev.Data[1]
			return tracker.PlayerProcessEvent{Frame: int(ev.DeltaFrames), On: true, Channel: int(channel), Note: note}, true
		default:
			// ignore all other MIDI messages
		}
	}
	return tracker.PlayerProcessEvent{}, false
}

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
		modelMessages := make(chan interface{}, 1024)
		playerMessages := make(chan tracker.PlayerMessage, 1024)
		model := tracker.NewModel(modelMessages, playerMessages)
		player := tracker.NewPlayer(cmd.DefaultService, playerMessages, modelMessages)
		tracker := gioui.NewTracker(model, cmd.DefaultService)
		tracker.SetInstrEnlarged(true) // start the vsti with the instrument editor enlarged
		go tracker.Main()
		context := VSTIProcessContext{make([]vst2.MIDIEvent, 100), h}
		buf := make([]float32, 2048)
		return vst2.Plugin{
				UniqueID:       PLUGIN_ID,
				Version:        version,
				InputChannels:  0,
				OutputChannels: 2,
				Name:           PLUGIN_NAME,
				Vendor:         "vsariola/sointu",
				Category:       vst2.PluginCategorySynth,
				Flags:          vst2.PluginIsSynth,
				ProcessFloatFunc: func(in, out vst2.FloatBuffer) {
					left := out.Channel(0)
					right := out.Channel(1)
					if len(buf) < out.Frames*2 {
						buf = append(buf, make([]float32, out.Frames*2-len(buf))...)
					}
					buf = buf[:out.Frames*2]
					player.Process(buf, &context)
					for i := 0; i < out.Frames; i++ {
						left[i], right[i] = buf[i*2], buf[i*2+1]
					}
					context.events = context.events[:0]
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
					tracker.Quit(true)
				},
			}

	}
}

func main() {}
