package vm

import (
	"math"
	"math/bits"
	"runtime"
	"sync"

	"github.com/vsariola/sointu"
)

type (
	ParallelSynth struct {
		voiceMapping [][]int
		synths       []sointu.Synth
		commands     chan<- parallelSynthCommand // maxtime
		results      <-chan parallelSynthResult  // rendered buffer
		pool         sync.Pool
		synther      sointu.Synther
	}

	ParallelSynther struct {
		synther sointu.Synther
		name    string
	}

	parallelSynthCommand struct {
		core    int
		samples int
		time    int
	}

	parallelSynthResult struct {
		buffer      *sointu.AudioBuffer
		samples     int
		time        int
		renderError error
	}
)

func MakeParallelSynther(synther sointu.Synther) ParallelSynther {
	return ParallelSynther{synther: synther, name: "Parallel " + synther.Name()}
}

func (s ParallelSynther) Name() string              { return s.name }
func (s ParallelSynther) SupportsParallelism() bool { return true }

func (s ParallelSynther) Synth(patch sointu.Patch, bpm int) (sointu.Synth, error) {
	patches, voiceMapping := splitPatchByCores(patch)
	synths := make([]sointu.Synth, 0, len(patches))
	for _, p := range patches {
		synth, err := s.synther.Synth(p, bpm)
		if err != nil {
			return nil, err
		}
		synths = append(synths, synth)
	}
	ret := &ParallelSynth{
		synths:       synths,
		voiceMapping: voiceMapping,
		pool:         sync.Pool{New: func() any { ret := make(sointu.AudioBuffer, 0, 8096); return &ret }},
	}
	ret.startProcesses()
	ret.synther = s.synther
	return ret, nil
}

func (s *ParallelSynth) Update(patch sointu.Patch, bpm int) error {
	patches, voiceMapping := splitPatchByCores(patch)
	s.voiceMapping = voiceMapping
	for i, p := range patches {
		if len(s.synths) <= i {
			synth, err := s.synther.Synth(p, bpm)
			if err != nil {
				return err
			}
			s.synths = append(s.synths, synth)
		} else {
			if err := s.synths[i].Update(p, bpm); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ParallelSynth) startProcesses() {
	maxProcs := runtime.GOMAXPROCS(0)
	cmdChan := make(chan parallelSynthCommand, maxProcs)
	s.commands = cmdChan
	resultsChan := make(chan parallelSynthResult, maxProcs)
	s.results = resultsChan
	for i := 0; i < maxProcs; i++ {
		go func(commandCh <-chan parallelSynthCommand, resultCh chan<- parallelSynthResult) {
			for cmd := range commandCh {
				buffer := s.pool.Get().(*sointu.AudioBuffer)
				*buffer = append(*buffer, make(sointu.AudioBuffer, cmd.samples)...)
				samples, time, renderError := s.synths[cmd.core].Render(*buffer, cmd.time)
				resultCh <- parallelSynthResult{buffer: buffer, samples: samples, time: time, renderError: renderError}
			}
		}(cmdChan, resultsChan)
	}
}

func (s *ParallelSynth) Close() {
	close(s.commands)
	for _, synth := range s.synths {
		synth.Close()
	}
}

func (s *ParallelSynth) Trigger(voiceIndex int, note byte) {
	for core, synth := range s.synths {
		if ind := s.voiceMapping[core][voiceIndex]; ind >= 0 {
			synth.Trigger(ind, note)
		}
	}
}

func (s *ParallelSynth) Release(voiceIndex int) {
	for core, synth := range s.synths {
		if ind := s.voiceMapping[core][voiceIndex]; ind >= 0 {
			synth.Release(ind)
		}
	}
}

func (s *ParallelSynth) Render(buffer sointu.AudioBuffer, maxtime int) (samples int, time int, renderError error) {
	count := len(s.synths)
	for i := 0; i < count; i++ {
		s.commands <- parallelSynthCommand{core: i, samples: len(buffer), time: maxtime}
	}
	clear(buffer)
	samples = math.MaxInt
	time = math.MaxInt
	for i := 0; i < count; i++ {
		result := <-s.results
		if result.renderError != nil && renderError == nil {
			renderError = result.renderError
		}
		samples = min(samples, result.samples)
		time = min(time, result.time)
		for j := 0; j < samples; j++ {
			buffer[j][0] += (*result.buffer)[j][0]
			buffer[j][1] += (*result.buffer)[j][1]
		}
		*result.buffer = (*result.buffer)[:0]
		s.pool.Put(result.buffer)
	}
	return
}

func splitPatchByCores(patch sointu.Patch) ([]sointu.Patch, [][]int) {
	maxCores := 1
	for _, instr := range patch {
		maxCores = max(bits.Len(instr.CoreBitMask), maxCores)
	}
	ret := make([]sointu.Patch, maxCores)
	for core := 0; core < maxCores; core++ {
		ret[core] = make(sointu.Patch, 0, len(patch))
	}
	voicemapping := make([][]int, maxCores)
	for core := range maxCores {
		voicemapping[core] = make([]int, patch.NumVoices())
		for j := range voicemapping[core] {
			voicemapping[core][j] = -1
		}
		coreVoice := 0
		curVoice := 0
		for _, instr := range patch {
			if instr.CoreBitMask == 0 || (instr.CoreBitMask&(1<<core)) != 0 {
				ret[core] = append(ret[core], instr)
				for j := 0; j < instr.NumVoices; j++ {
					voicemapping[core][curVoice+j] = coreVoice + j
				}
				coreVoice += instr.NumVoices
			}
			curVoice += instr.NumVoices
		}
	}
	return ret, voicemapping
}
