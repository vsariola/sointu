(module

{{- /*
;------------------------------------------------------------------------------
;    Patterns
;-------------------------------------------------------------------------------
*/}}
{{- .SetDataLabel "su_patterns"}}
{{- $m := .}}
{{- range .Patterns}}
    {{- range .}}
        {{- $.DataB .}}
    {{- end}}
{{- end}}

{{- /*
;------------------------------------------------------------------------------
;    Tracks
;-------------------------------------------------------------------------------
*/}}
{{- .SetDataLabel "su_tracks"}}
{{- $m := .}}
{{- range .Sequences}}
    {{- range .}}
        {{- $.DataB .}}
    {{- end}}
{{- end}}

{{- /*
;------------------------------------------------------------------------------
;    The code for this patch, basically indices to vm jump table
;-------------------------------------------------------------------------------
*/}}
{{- .SetDataLabel "su_patch_code"}}
{{- range .Commands}}
{{- $.DataB .}}
{{- end}}

{{- /*
;-------------------------------------------------------------------------------
;    The parameters / inputs to each opcode
;-------------------------------------------------------------------------------
*/}}
{{- .SetDataLabel "su_patch_parameters"}}
{{- range .Values}}
{{- $.DataB .}}
{{- end}}

{{- /*
;-------------------------------------------------------------------------------
;    Delay times
;-------------------------------------------------------------------------------
*/}}
{{- .SetDataLabel "su_delay_times"}}
{{- range .DelayTimes}}
{{- $.DataW .}}
{{- end}}

{{- /*
;-------------------------------------------------------------------------------
; The number of transformed parameters each opcode takes
;-------------------------------------------------------------------------------
*/}}
{{- .SetDataLabel "su_vm_transformcounts"}}
{{- range .Instructions}}
{{- $.TransformCount . | $.ToByte | $.DataB}}
{{- end}}

{{- /*
;-------------------------------------------------------------------------------
; Allocate memory for stack.
; Stack of 64 float signals is enough for everybody... right?
; Note: as the stack grows _downwards_ the label is _after_ stack
;-------------------------------------------------------------------------------
*/}}
{{- .Align}}
{{- .Block 256}}
{{- .SetBlockLabel "su_stack"}}

{{- /*
;-------------------------------------------------------------------------------
; Allocate memory for transformed values.
;-------------------------------------------------------------------------------
*/}}
{{- .Align}}
{{- .SetBlockLabel "su_transformedvalues"}}
{{- .Block 32}}

{{- /*
;-------------------------------------------------------------------------------
; Uninitialized memory for synth, delaylines & outputbuffer
;-------------------------------------------------------------------------------
*/}}
{{- .Align}}
{{- if ne .VoiceTrackBitmask 0}}
{{- .SetBlockLabel "su_trackcurrentvoice"}}
{{- .Block 32}}
{{- end}}
{{- .Align}}
{{- .SetBlockLabel "su_synth"}}
{{- .Block 32}}
{{- .SetBlockLabel "su_globalports"}}
{{- .Block 32}}
{{- .SetBlockLabel "su_voices"}}
{{- .Block 131072}}
{{- .Align}}
{{- .SetBlockLabel "su_delaylines"}}
{{- .Block (int (mul 262156 .Song.Patch.NumDelayLines))}}
{{- .Align}}
{{- .SetBlockLabel "su_outputbuffer"}}
{{- if .Output16Bit}}
{{- .Block (int (mul .PatternLength .SequenceLength .Song.SamplesPerRow 4))}}
{{- else}}
{{- .Block (int (mul .PatternLength .SequenceLength .Song.SamplesPerRow 8))}}
{{- end}}
{{- .SetBlockLabel "su_outputend"}}


;;------------------------------------------------------------------------------
;; Import the difficult math functions from javascript
;; (seriously now, it's 2020)
;;------------------------------------------------------------------------------
(func $pow (import "m" "pow") (param f32) (param f32) (result f32))
(func $log2 (import "m" "log2") (param f32) (result f32))
(func $sin (import "m" "sin") (param f32) (result f32))

;;------------------------------------------------------------------------------
;; Types. Only useful to define the jump table type, which is
;; (int stereo) void
;;------------------------------------------------------------------------------
(type $opcode_func_signature (func (param i32)))

;;------------------------------------------------------------------------------
;; The one and only memory
;;------------------------------------------------------------------------------
(memory (export "m") {{.MemoryPages}})

;;------------------------------------------------------------------------------
;; Globals. Putting all with same initialization value should compress most
;;------------------------------------------------------------------------------
(global $WRK (mut i32) (i32.const 0))
(global $COM (mut i32) (i32.const 0))
(global $VAL (mut i32) (i32.const 0))
{{- if .SupportsPolyphony}}
(global $COM_instr_start (mut i32) (i32.const 0))
(global $VAL_instr_start (mut i32) (i32.const 0))
{{- end}}
{{- if .HasOp "delay"}}
(global $delayWRK (mut i32) (i32.const 0))
{{- end}}
(global $globaltick (mut i32) (i32.const 0))
(global $row (mut i32) (i32.const 0))
(global $pattern (mut i32) (i32.const 0))
(global $sample (mut i32) (i32.const 0))
(global $voice (mut i32) (i32.const 0))
(global $voicesRemain (mut i32) (i32.const 0))
(global $randseed (mut i32) (i32.const 1))
(global $sp (mut i32) (i32.const {{index .Labels "su_stack"}}))
(global $outputBufPtr (mut i32) (i32.const {{index .Labels "su_outputbuffer"}}))
;; TODO: only export start and length with certain compiler options; in demo use, they can be hard coded
;; in the intro
(global $outputStart (export "s") i32 (i32.const {{index .Labels "su_outputbuffer"}}))
(global $outputLength (export "l") i32 (i32.const {{if .Output16Bit}}{{mul .PatternLength .SequenceLength .Song.SamplesPerRow 4}}{{else}}{{mul .PatternLength .SequenceLength .Song.SamplesPerRow 8}}{{end}}))
(global $output16bit (export "t") i32 (i32.const {{if .Output16Bit}}1{{else}}0{{end}}))


;;------------------------------------------------------------------------------
;; Functions to emulate FPU stack in software
;;------------------------------------------------------------------------------
(func $peek (result f32)
    (f32.load (global.get $sp))
)

(func $peek2 (result f32)
    (f32.load offset=4 (global.get $sp))
)

(func $pop (result f32)
    (call $peek)
    (global.set $sp (i32.add (global.get $sp) (i32.const 4)))
)

(func $push (param $value f32)
    (global.set $sp (i32.sub (global.get $sp) (i32.const 4)))
    (f32.store (global.get $sp) (local.get $value))
)

;;------------------------------------------------------------------------------
;; Helper functions
;;------------------------------------------------------------------------------
(func $swap (param f32 f32) (result f32 f32) ;; x,y -> y,x
    local.get 1
    local.get 0
)

(func $scanValueByte (result i32)        ;; scans positions $VAL for a byte, incrementing $VAL afterwards
    (i32.load8_u (global.get $VAL))      ;; in other words: returns byte [$VAL++]
    (global.set $VAL (i32.add (global.get $VAL) (i32.const 1))) ;; $VAL++
)

;;------------------------------------------------------------------------------
;; "Entry point" for the player
;;------------------------------------------------------------------------------
(start $render) ;; we run render automagically when the module is instantiated

(func $render (param)
{{- if  .Output16Bit }} (local $channel i32) {{- end }}
    loop $pattern_loop
        (global.set $row (i32.const 0))
        loop $row_loop
            (call $su_update_voices)
            (global.set $sample (i32.const 0))
            loop $sample_loop
                (global.set $COM (i32.const {{index .Labels "su_patch_code"}}))
                (global.set $VAL (i32.const {{index .Labels "su_patch_parameters"}}))
{{- if .SupportsPolyphony}}
                (global.set $COM_instr_start (global.get $COM))
                (global.set $VAL_instr_start (global.get $VAL))
{{- end}}
                (global.set $WRK (i32.const {{index .Labels "su_voices"}}))
                (global.set $voice (i32.const {{index .Labels "su_voices"}}))
                (global.set $voicesRemain (i32.const {{.Song.Patch.NumVoices | printf "%v"}}))
{{- if .HasOp "delay"}}
                (global.set $delayWRK (i32.const {{index .Labels "su_delaylines"}}))
{{- end}}
                (call $su_run_vm)
                {{- template "output_sound.wat" .}}
                (global.set $sample (i32.add (global.get $sample) (i32.const 1)))
                (global.set $globaltick (i32.add (global.get $globaltick) (i32.const 1)))
                (br_if $sample_loop (i32.lt_s (global.get $sample) (i32.const {{.Song.SamplesPerRow}})))
            end
            (global.set $row (i32.add (global.get $row) (i32.const 1)))
            (br_if $row_loop (i32.lt_s (global.get $row) (i32.const {{.PatternLength}})))
        end
        (global.set $pattern (i32.add (global.get $pattern) (i32.const 1)))
        (br_if $pattern_loop (i32.lt_s (global.get $pattern) (i32.const {{.SequenceLength}})))
    end
)

{{- if ne .VoiceTrackBitmask 0}}
;; the complex implementation of update_voices: at least one track has more than one voice
(func $su_update_voices (local $si i32) (local $di i32) (local $tracksRemaining i32) (local $note i32) (local $firstVoice i32) (local $nextTrackStartsAt i32) (local $numVoices i32) (local $voiceNo i32)
    (local.set $tracksRemaining (i32.const {{len .Sequences}}))
    (local.set $si (global.get $pattern))
    (local.set $nextTrackStartsAt (i32.const 0))
    loop $track_loop
        (local.set $numVoices (i32.const 0))
        (local.set $firstVoice (local.get $nextTrackStartsAt))
        loop $voiceLoop
            (i32.and
                (i32.shr_u
                    (i32.const {{.VoiceTrackBitmask | printf "%v"}})
                    (local.get $nextTrackStartsAt)
                )
                (i32.const 1)
            )
            (local.set $nextTrackStartsAt (i32.add (local.get $nextTrackStartsAt) (i32.const 1)))
            (local.set $numVoices (i32.add (local.get $numVoices) (i32.const 1)))
            br_if $voiceLoop
        end
        (i32.load8_u offset={{index .Labels "su_tracks"}} (local.get $si))
        (i32.mul (i32.const {{.PatternLength}}))
        (i32.add (global.get $row))
        (i32.load8_u offset={{index .Labels "su_patterns"}})
        (local.tee $note)
        (if (i32.ne (i32.const {{.Hold}}))(then
            (i32.store offset={{add (index .Labels "su_voices") 4}}
                (i32.mul
                    (i32.add
                        (local.tee $voiceNo (i32.load8_u offset={{index .Labels "su_trackcurrentvoice"}} (local.get $tracksRemaining)))
                        (local.get $firstVoice)
                    )
                    (i32.const 4096)
                )
                (i32.const 0)
            ) ;; release the note
            (if (i32.gt_u (local.get $note) (i32.const {{.Hold}}))(then
                (local.set $di (i32.add
                    (i32.mul
                        (i32.add
                            (local.tee $voiceNo (i32.rem_u
                                (i32.add (local.get $voiceNo) (i32.const 1))
                                (local.get $numVoices)
                            ))
                            (local.get $firstVoice)
                        )
                        (i32.const 4096)
                    )
                    (i32.const {{index .Labels "su_voices"}})
                ))
                (memory.fill (local.get $di) (i32.const 0) (i32.const 4096))
                (i32.store (local.get $di) (local.get $note))
                (i32.store offset=4 (local.get $di) (local.get $note))
                (i32.store8 offset={{index .Labels "su_trackcurrentvoice"}} (local.get $tracksRemaining) (local.get $voiceNo))
            ))
        ))
        (local.set $si (i32.add (local.get $si) (i32.const {{.SequenceLength}})))
        (br_if $track_loop (local.tee $tracksRemaining (i32.sub (local.get $tracksRemaining) (i32.const 1))))
    end
)

{{- else}}
;; the simple implementation of update_voices: each track has exactly one voice
(func $su_update_voices (local $si i32) (local $di i32) (local $tracksRemaining i32) (local $note i32)
    (local.set $tracksRemaining (i32.const {{len .Sequences}}))
    (local.set $si (global.get $pattern))
    (local.set $di (i32.const {{index .Labels "su_voices"}}))
    loop $track_loop
        (i32.load8_u offset={{index .Labels "su_tracks"}} (local.get $si))
        (i32.mul (i32.const {{.PatternLength}}))
        (i32.add (global.get $row))
        (i32.load8_u offset={{index .Labels "su_patterns"}})
        (local.tee $note)
        (if (i32.ne (i32.const {{.Hold}}))(then
            (i32.store offset=4 (local.get $di) (i32.const 0)) ;; release the note
            (if (i32.gt_u (local.get $note) (i32.const {{.Hold}}))(then
                (memory.fill (local.get $di) (i32.const 0) (i32.const 4096))
                (i32.store (local.get $di) (local.get $note))
                (i32.store offset=4 (local.get $di) (local.get $note))
            ))
        ))
        (local.set $di (i32.add (local.get $di) (i32.const 4096)))
        (local.set $si (i32.add (local.get $si) (i32.const {{.SequenceLength}})))
        (br_if $track_loop (local.tee $tracksRemaining (i32.sub (local.get $tracksRemaining) (i32.const 1))))
    end
)
{{- end}}

{{template "patch.wat" .}}


;; All data is collected into a byte buffer and emitted at once
(data (i32.const 0) "{{range .Data}}\{{. | printf "%02x"}}{{end}}")

;;(data (i32.const 8388610) "\52\49\46\46\b2\eb\0c\20\57\41\56\45\66\6d\74\20\12\20\20\20\03\20\02\20\44\ac\20\20\20\62\05\20\08\20\20\20\20\20\66\61\63\74\04\20\20\20\e0\3a\03\20\64\61\74\61\80\eb\0c\20")

) ;; END MODULE
