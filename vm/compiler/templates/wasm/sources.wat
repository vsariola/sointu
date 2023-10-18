{{- if .HasOp "loadval"}}
;;-------------------------------------------------------------------------------
;;   LOADVAL opcode
;;-------------------------------------------------------------------------------
{{- if .Mono "loadval"}}
;;   Mono: push 2*v-1 on stack, where v is the input to port "value"
{{- end}}
{{- if .Stereo "loadval"}}
;;   Stereo: push 2*v-1 twice on stack
{{- end}}
;;-------------------------------------------------------------------------------
(func $su_op_loadval (param $stereo i32)
{{- if .Stereo "loadval"}}
    (if (local.get $stereo) (then
        (call $su_op_loadval (i32.const 0))
    ))
{{- end}}
    (f32.sub (call $input (i32.const {{.InputNumber "loadval" "value"}})) (f32.const 0.5))
    (f32.mul (f32.const 2.0))
    (call $push)
)
{{end}}


{{if .HasOp "envelope" -}}
;;-------------------------------------------------------------------------------
;;   ENVELOPE opcode: pushes an ADSR envelope value on stack [0,1]
;;-------------------------------------------------------------------------------
;;   Mono:   push the envelope value on stack
;;   Stereo: push the envelope valeu on stack twice
;;-------------------------------------------------------------------------------
(func $su_op_envelope (param $stereo i32) (local $state i32) (local $level f32) (local $delta f32)
    (if (i32.eqz (i32.load offset=4 (global.get $voice))) (then ;; if voice.sustain == 0
        (i32.store (global.get $WRK) (i32.const {{.InputNumber "envelope" "release"}})) ;; set envelope state to release
    ))
    (local.set $state (i32.load (global.get $WRK)))
    (local.set $level (f32.load offset=4 (global.get $WRK)))
    (local.set $delta (call $nonLinearMap (local.get $state)))
    (if (local.get $state) (then
        (if (i32.eq (local.get $state) (i32.const 1))(then ;; state is 1 aka decay
            (local.set $level (f32.sub (local.get $level) (local.get $delta)))
            (if (f32.le (local.get $level) (call $input (i32.const 2)))(then
                (local.set $level (call $input (i32.const 2)))
                (local.set $state (i32.const {{.InputNumber "envelope" "sustain"}}))
            ))
        ))
        (if (i32.eq (local.get $state) (i32.const {{.InputNumber "envelope" "release"}}))(then ;; state is 3 aka release
            (local.set $level (f32.sub (local.get $level) (local.get $delta)))
            (if (f32.le (local.get $level) (f32.const 0)) (then
                (local.set $level (f32.const 0))
            ))
        ))
    )(else ;; the state is 0 aka attack
        (local.set $level (f32.add (local.get $level) (local.get $delta)))
        (if (f32.ge (local.get $level) (f32.const 1))(then
            (local.set $level (f32.const 1))
            (local.set $state (i32.const 1))
        ))
    ))
    (i32.store (global.get $WRK) (local.get $state))
    (f32.store offset=4 (global.get $WRK) (local.get $level))
    (call $push (f32.mul (local.get $level) (call $input (i32.const {{.InputNumber "envelope" "gain"}}))))
{{- if .Stereo "envelope"}}
    (if (local.get $stereo)(then
        (call $push (call $peek))
    ))
{{- end}}
)
{{end}}


{{- if .HasOp "noise"}}
;;-------------------------------------------------------------------------------
;;   NOISE opcode: creates noise
;;-------------------------------------------------------------------------------
;;   Mono:   push a random value [-1,1] value on stack
;;   Stereo: push two (different) random values on stack
;;-------------------------------------------------------------------------------
(func $su_op_noise (param $stereo i32)
{{- if .Stereo "noise" }}
    (if (local.get $stereo) (then
        (call $su_op_noise (i32.const 0))
    ))
{{- end}}
    (global.set $randseed (i32.mul (global.get $randseed) (i32.const 16007)))
    (f32.mul
        (call $waveshaper
            ;; Note: in x86 code, the constant looks like a positive integer, but has actually the MSB set i.e. is considered negative by the FPU. This tripped me big time.
            (f32.div (f32.convert_i32_s (global.get $randseed)) (f32.const -2147483648))
            (call $input (i32.const {{.InputNumber "noise" "shape"}}))
        )
        (call $input (i32.const {{.InputNumber "noise" "gain"}}))
    )
    (call $push)
)
{{end}}


{{- if .HasOp "oscillator"}}
;;-------------------------------------------------------------------------------
;;   OSCILLAT opcode: oscillator, the heart of the synth
;;-------------------------------------------------------------------------------
;;   Mono:   push oscillator value on stack
;;   Stereo: push l r on stack, where l has opposite detune compared to r
;;-------------------------------------------------------------------------------
(func $su_op_oscillator (param $stereo i32) (local $flags i32) (local $detune f32) (local $phase f32) (local $color f32) (local $amplitude f32)
{{- if .SupportsParamValueOtherThan "oscillator" "unison" 0}}
    (local $unison i32) (local $WRK_stash i32) (local $detune_stash f32)
{{- end}}
{{- if .SupportsModulation "oscillator" "frequency"}}
    (local $freqMod f32)
{{- end}}
{{- if .Stereo "oscillator"}}
    (local $WRK_stereostash i32)
    (local.set $WRK_stereostash (global.get $WRK))
{{- end}}
{{- if .SupportsModulation "oscillator" "frequency"}}
    (local.set $freqMod (f32.load offset={{.InputNumber "oscillator" "frequency" | mul 4 | add 32}} (global.get $WRK)))
    (f32.store offset={{.InputNumber "oscillator" "frequency" | mul 4 | add 32}} (global.get $WRK) (f32.const 0))
{{- end}}
    (local.set $flags (call $scanOperand))
    (local.set $detune (call $inputSigned (i32.const {{.InputNumber "oscillator" "detune"}})))
{{- if .Stereo "oscillator"}}
    loop $stereoLoop
{{- end}}
{{- if .SupportsParamValueOtherThan "oscillator" "unison" 0}}
    (local.set $unison (i32.add (i32.and (local.get $flags) (i32.const 3)) (i32.const 1)))
    (local.set $WRK_stash (global.get $WRK))
    (local.set $detune_stash (local.get $detune))
    (call $push (f32.const 0))
    loop $unisonLoop
{{- end}}
    (f32.store ;; update phase
        (global.get $WRK)
        (local.tee $phase
            (f32.sub
                (local.tee $phase
                    ;; Transpose calculation starts
                    (f32.div
                        (call $inputSigned (i32.const {{.InputNumber "oscillator" "transpose"}}))
                        (f32.const 0.015625)
                    ) ;; scale back to 0 - 128
                    (f32.add (local.get $detune)) ;; add detune. detune is -1 to 1 so can detune a full note up or down at max
                    (f32.add (select
                        (f32.const 0)
                        (f32.convert_i32_u (i32.load (global.get $voice)))
                        (i32.and (local.get $flags) (i32.const 0x8))
                    ))  ;; if lfo is not enabled, add the note number to it
                    (f32.mul (f32.const 0.0833333)) ;; /12, in full octaves
                    (call $pow2)
                    (f32.mul (select
                        (f32.const 0.000038) ;; pretty random scaling constant to get LFOs into reasonable range. Historical reasons, goes all the way back to 4klang
                        (f32.const 0.000092696138) ;; scaling constant to get middle-C to where it should be
                        (i32.and (local.get $flags) (i32.const 0x8))
                    ))
{{- if .SupportsModulation "oscillator" "frequency"}}
                    (f32.add (local.get $freqMod))
{{- end}}
                    (f32.add (f32.load (global.get $WRK))) ;; add the current phase of the oscillator
                )
                (f32.floor (local.get $phase))
            )
        )
    )
    (f32.add (local.get $phase) (call $input (i32.const {{.InputNumber "oscillator" "phase"}})))
    (local.set $phase (f32.sub (local.tee $phase) (f32.floor (local.get $phase)))) ;; phase = phase mod 1.0
    (local.set $color (call $input (i32.const {{.InputNumber "oscillator" "color"}})))
{{- if .SupportsParamValue "oscillator" "type" .Sine}}
    (if (i32.and (local.get $flags) (i32.const 0x40)) (then
        (local.set $amplitude (call $oscillator_sine (local.get $phase) (local.get $color)))
    ))
{{- end}}
{{- if .SupportsParamValue "oscillator" "type" .Trisaw}}
    (if (i32.and (local.get $flags) (i32.const 0x20)) (then
        (local.set $amplitude (call $oscillator_trisaw (local.get $phase) (local.get $color)))
    ))
{{- end}}
{{- if .SupportsParamValue "oscillator" "type" .Pulse}}
    (if (i32.and (local.get $flags) (i32.const 0x10)) (then
        (local.set $amplitude (call $oscillator_pulse (local.get $phase) (local.get $color)))
    ))
{{- end}}
{{- if .SupportsParamValue "oscillator" "type" .Gate}}
    (if (i32.and (local.get $flags) (i32.const 0x04)) (then
        (local.set $amplitude (call $oscillator_gate (local.get $phase)))
        ;; wave shaping is skipped with gate
    )(else
        (local.set $amplitude (call $waveshaper (local.get $amplitude) (call $input (i32.const {{.InputNumber "oscillator" "shape"}}))))
    ))
    (local.get $amplitude)
{{- else}}
    (call $waveshaper (local.get $amplitude) (call $input (i32.const {{.InputNumber "oscillator" "shape"}})))
{{- end}}
    (call $push (f32.mul
        (call $input (i32.const {{.InputNumber "oscillator" "gain"}}))
    ))
{{- if .SupportsParamValueOtherThan "oscillator" "unison" 0}}
    (call $push (f32.add (call $pop) (call $pop)))
    (if (local.tee $unison (i32.sub (local.get $unison) (i32.const 1)))(then
        (f32.store offset={{.InputNumber "oscillator" "phase" | mul 4 | add (index .Labels "su_transformedoperands")}} (i32.const 0)
            (f32.add
                (call $input (i32.const {{.InputNumber "oscillator" "phase"}}))
                (f32.const 0.08333333) ;; 1/12, add small phase shift so all oscillators don't start in phase
            )
        )
        (global.set $WRK (i32.add (global.get $WRK) (i32.const 8)))
        (local.set $detune (f32.neg (f32.mul
            (local.get $detune) ;; each unison oscillator has a detune with flipped sign and halved amount... this creates detunes that concentrate around the fundamental
            (f32.const 0.5)
        )))
        br $unisonLoop
    ))
    end
    (global.set $WRK (local.get $WRK_stash))
    (local.set $detune (local.get $detune_stash))
{{- end}}
{{- if .Stereo "oscillator"}}
    (local.set $detune (f32.neg (local.get $detune))) ;; flip the detune for secon round
    (global.set $WRK (i32.add (global.get $WRK) (i32.const 4)))
    (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
    (global.set $WRK (local.get $WRK_stereostash))
    ;; TODO: all this "save WRK to local variable, modify it and then restore it" could be better thought out
    ;; however, it is now done like this as a quick bug fix to the issue of stereo oscillators touching WRK and not restoring it
{{- end}}
)

{{- if .SupportsParamValue "oscillator" "type" .Pulse}}
(func $oscillator_pulse (param $phase f32) (param $color f32) (result f32)
    (select
        (f32.const -1)
        (f32.const 1)
        (f32.ge (local.get $phase) (local.get $color))
    )
)
{{end}}

{{- if .SupportsParamValue "oscillator" "type" .Sine}}
(func $oscillator_sine (param $phase f32) (param $color f32) (result f32)
    (select
        (f32.const 0)
        (call $sin (f32.mul
            (f32.div
                (local.get $phase)
                (local.get $color)
            )
            (f32.const 6.28318530718)
        ))
        (f32.ge (local.get $phase) (local.get $color))
    )
)
{{end}}

{{- if .SupportsParamValue "oscillator" "type" .Trisaw}}
(func $oscillator_trisaw (param $phase f32) (param $color f32) (result f32)
    (if (f32.ge (local.get $phase) (local.get $color)) (then
        (local.set $phase (f32.sub (f32.const 1) (local.get $phase)))
        (local.set $color (f32.sub (f32.const 1) (local.get $color)))
    ))
    (f32.div (local.get $phase) (local.get $color))
    (f32.mul (f32.const 2))
    (f32.sub (f32.const 1))
)
{{end}}

{{- if .SupportsParamValue "oscillator" "type" .Gate}}
(func $oscillator_gate (param $phase f32) (result f32) (local $x f32)
    (f32.store offset=16 (global.get $WRK)
        (local.tee $x
            (f32.add ;; c*(g-x)+x
                (f32.mul ;; c*(g-x)
                    (f32.sub ;; g - x
                        (f32.load offset=16 (global.get $WRK)) ;; g
                        (local.tee $x
                            (f32.convert_i32_u ;; 'x' gate bit = float((gatebits >> (int(p*16+.5)&15)) & 1)
                                (i32.and ;; (int(p*16+.5)&15)&1
                                    (i32.shr_u ;; int(p*16+.5)&15
                                        (i32.load16_u (i32.sub (global.get $VAL) (i32.const 4)))
                                        (i32.and ;; int(p*16+.5) & 15
                                            (i32.trunc_f32_s (f32.add
                                                (f32.mul
                                                    (local.get $phase)
                                                    (f32.const 16.0)
                                                )
                                                (f32.const 0.5) ;; well, x86 rounds to integer by default; on wasm, we have only trunc.
                                            ))                  ;; This is just for rendering similar to x86, should probably delete when optimizing size.
                                            (i32.const 15)
                                        )
                                    )
                                    (i32.const 1)
                                )
                            )
                        )
                    )
                    (f32.const 0.99609375) ;; 'c'
                )
                (local.get $x)
            )
        )
    )
    local.get $x
)
{{end}}

{{end}}


{{- if .HasOp "receive"}}
;;-------------------------------------------------------------------------------
;;   RECEIVE opcode
;;-------------------------------------------------------------------------------
{{- if .Mono "receive"}}
;;   Mono:   push l on stack, where l is the left channel received
{{- end}}
{{- if .Stereo "receive"}}
;;   Stereo: push l r on stack
{{- end}}
;;-------------------------------------------------------------------------------
(func $su_op_receive (param $stereo i32)
{{- if .Stereo "receive"}}
    (if (local.get $stereo) (then
        (call $push
            (f32.load offset=36 (global.get $WRK))
        )
        (f32.store offset=36 (global.get $WRK) (f32.const 0))
    ))
{{- end}}
    (call $push
        (f32.load offset=32 (global.get $WRK))
    )
    (f32.store offset=32 (global.get $WRK) (f32.const 0))
)
{{end}}


{{- if .HasOp "in"}}
;;-------------------------------------------------------------------------------
;;   IN opcode: inputs and clears a global port
;;-------------------------------------------------------------------------------
;;   Mono: push the left channel of a global port (out or aux)
;;   Stereo: also push the right channel (stack in l r order)
;;-------------------------------------------------------------------------------
(func $su_op_in (param $stereo i32) (local $addr i32)
    call $scanOperand
{{- if .Stereo "in"}}
    (i32.add (local.get $stereo)) ;; start from right channel if stereo
{{- end}}
    (local.set $addr (i32.add (i32.mul (i32.const 4)) (i32.const {{index .Labels "su_globalports"}})))
{{- if .Stereo "in"}}
    loop $stereoLoop
{{- end}}
        (call $push (f32.load (local.get $addr)))
        (f32.store (local.get $addr) (f32.const 0))
{{- if .Stereo "in"}}
        (local.set $addr (i32.sub (local.get $addr) (i32.const 4)))
        (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
{{- end}}
)
{{end}}
