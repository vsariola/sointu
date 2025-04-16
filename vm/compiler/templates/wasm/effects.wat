{{- if .HasOp "distort"}}
;;-------------------------------------------------------------------------------
;;   DISTORT opcode: apply distortion on the signal
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  x*a/(1-a+(2*a-1)*abs(x))            where x is clamped first
;;   Stereo: l r ->  l*a/(1-a+(2*a-1)*abs(l)) r*a/(1-a+(2*a-1)*abs(r))
;;-------------------------------------------------------------------------------
(func $su_op_distort (param $stereo i32)
{{- if .Stereo "distort"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "distort") 2}}))
{{- end}}
    (call $pop)
    (call $waveshaper (call $input (i32.const {{.InputNumber "distort" "drive"}})))
    (call $push)
)
{{end}}


{{- if .HasOp "hold"}}
;;-------------------------------------------------------------------------------
;;   HOLD opcode: sample and hold the signal, reducing sample rate
;;-------------------------------------------------------------------------------
;;   Mono version:   holds the signal at a rate defined by the freq parameter
;;   Stereo version: holds both channels
;;-------------------------------------------------------------------------------
(func $su_op_hold (param $stereo i32) (local $phase f32)
{{- if .Stereo "hold"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "hold") 2}}))
{{- end}}
    (local.set $phase
        (f32.sub
            (f32.load (global.get $WRK))
            (f32.mul
                (call $input (i32.const {{.InputNumber "hold" "holdfreq"}}))
                (call $input (i32.const {{.InputNumber "hold" "holdfreq"}})) ;; if we ever implement $dup, replace with that
            )
        )
    )
    (if (f32.ge (f32.const 0) (local.get $phase)) (then
        (f32.store offset=4 (global.get $WRK) (call $peek)) ;; we start holding a new value
        (local.set $phase (f32.add (local.get $phase) (f32.const 1)))
    ))
    (drop (call $pop))                                 ;; we replace the top most signal
    (call $push (f32.load offset=4 (global.get $WRK))) ;; with the held value
    (f32.store (global.get $WRK) (local.get $phase)) ;; save back new phase
)
{{end}}


{{- if .HasOp "crush"}}
;;-------------------------------------------------------------------------------
;;   CRUSH opcode: quantize the signal to finite number of levels
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  e*int(x/e)
;;   Stereo: l r ->  e*int(l/e) e*int(r/e)
;;-------------------------------------------------------------------------------
(func $su_op_crush (param $stereo i32) (local $e f32)
{{- if .Stereo "crush"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "crush") 2}}))
{{- end}}
    call $pop
    (f32.div (local.tee $e (call $nonLinearMap (i32.const {{.InputNumber "crush" "resolution"}}))))
    f32.nearest
    (f32.mul (local.get $e))
    call $push
)
{{end}}


{{- if .HasOp "gain"}}
;;-------------------------------------------------------------------------------
;;   GAIN opcode: apply gain on the signal
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  x*g
;;   Stereo: l r ->  l*g r*g
;;-------------------------------------------------------------------------------
(func $su_op_gain (param $stereo i32)
{{- if .Stereo "gain"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "gain") 2}}))
{{- end}}
    (call $push (f32.mul (call $pop) (call $input (i32.const {{.InputNumber "gain" "gain"}}))))
)
{{end}}


{{- if .HasOp "invgain"}}
;;-------------------------------------------------------------------------------
;;   INVGAIN opcode: apply inverse gain on the signal
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  x/g
;;   Stereo: l r ->  l/g r/g
;;-------------------------------------------------------------------------------
(func $su_op_invgain (param $stereo i32)
{{- if .Stereo "invgain"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "invgain") 2}}))
{{- end}}
    (call $push (f32.div (call $pop) (call $input (i32.const {{.InputNumber "invgain" "invgain"}}))))
)
{{end}}

{{- if .HasOp "dbgain"}}
;;-------------------------------------------------------------------------------
;;   DBGAIN opcode: apply gain on the signal, with gain given in decibels
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  x*g, where g = 2**((2*d-1)*6.643856189774724) i.e. -40dB to 40dB, d=[0..1]
;;   Stereo: l r ->  l*g r*g
;;-------------------------------------------------------------------------------
(func $su_op_dbgain (param $stereo i32)
{{- if .Stereo "dbgain"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "dbgain") 2}}))
{{- end}}
    (call $input (i32.const {{.InputNumber "dbgain" "decibels"}}))
    (f32.sub (f32.const 0.5))
    (f32.mul (f32.const 13.287712379549449))
    (call $pow2)
    (f32.mul (call $pop))
    (call $push)
)
{{end}}


{{- if .HasOp "filter"}}
;;-------------------------------------------------------------------------------
;;   FILTER opcode: perform low/high/band-pass/notch etc. filtering on the signal
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  filtered(x)
;;   Stereo: l r ->  filtered(l) filtered(r)
;;-------------------------------------------------------------------------------
(func $su_op_filter (param $stereo i32) (local $flags i32) (local $freq f32) (local $high f32) (local $low f32) (local $band f32) (local $retval f32)
{{- if .Stereo "filter"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "filter") 2}}))
    (if (local.get $stereo)(then
        ;; This is hacky: rewind the $VAL one byte backwards as the right channel already
        ;; scanned it once. Find a way to avoid rewind
        (global.set $VAL (i32.sub (global.get $VAL) (i32.const 1)))
    ))
{{- end}}
    (local.set $flags (call $scanOperand))
    (local.set $freq (f32.mul
        (call $input (i32.const {{.InputNumber "filter" "frequency"}}))
        (call $input (i32.const {{.InputNumber "filter" "frequency"}}))
    ))
    (local.set $low ;; l' = f2*b + l
        (f32.add ;; f2*b+l
            (f32.mul ;; f2*b
                (local.tee $band (f32.load offset=4 (global.get $WRK))) ;; b
                (local.get $freq)                     ;; f2
            )
            (f32.load (global.get $WRK)) ;; l
        )
    )
    (local.set $high ;; h' = x - l' - r*b
        (f32.sub ;; x - l' - r*b
            (f32.sub ;; x - l'
                (call $pop)      ;; x (signal)
                (local.get $low) ;; l'
            )
            (f32.mul ;; r*b
                (call $input (i32.const {{.InputNumber "filter" "resonance"}})) ;; r
                (local.get $band) ;; b
            )
        )
    )
    (local.set $band ;; b' = f2 * h' + b
        (f32.add ;; f2 * h' +  b
            (f32.mul ;; f2 * h'
                (local.get $freq) ;; f2
                (local.get $high) ;; h'
            )
            (local.get $band) ;; b
        )
    )
    (local.set $retval (f32.const 0))
{{- if .SupportsParamValue "filter" "lowpass" 1}}
    (if (i32.and (local.get $flags) (i32.const 0x40)) (then
        (local.set $retval (f32.add (local.get $retval) (local.get $low)))
    ))
{{- end}}
{{- if .SupportsParamValue "filter" "bandpass" 1}}
    (if (i32.and (local.get $flags) (i32.const 0x20)) (then
        (local.set $retval (f32.add (local.get $retval) (local.get $band)))
    ))
{{- end}}
{{- if .SupportsParamValue "filter" "highpass" 1}}
    (if (i32.and (local.get $flags) (i32.const 0x10)) (then
        (local.set $retval (f32.add (local.get $retval) (local.get $high)))
    ))
{{- end}}
{{- if .SupportsParamValue "filter" "negbandpass" 1}}
    (if (i32.and (local.get $flags) (i32.const 0x08)) (then
        (local.set $retval (f32.sub (local.get $retval) (local.get $band)))
    ))
{{- end}}
{{- if .SupportsParamValue "filter" "neghighpass" 1}}
    (if (i32.and (local.get $flags) (i32.const 0x04)) (then
        (local.set $retval (f32.sub (local.get $retval) (local.get $high)))
    ))
{{- end}}
    (f32.store (global.get $WRK) (local.get $low))
    (f32.store offset=4 (global.get $WRK) (local.get $band))
    (call $push (local.get $retval))
)
{{end}}


{{- if .HasOp "clip"}}
;;-------------------------------------------------------------------------------
;;   CLIP opcode: clips the signal into [-1,1] range
;;-------------------------------------------------------------------------------
;;   Mono:   x   ->  min(max(x,-1),1)
;;   Stereo: l r ->  min(max(l,-1),1) min(max(r,-1),1)
;;-------------------------------------------------------------------------------
(func $su_op_clip (param $stereo i32)
{{- if .Stereo "clip"}}
    (call $stereoHelper (local.get $stereo) (i32.const {{div (.GetOp "clip") 2}}))
{{- end}}
    (call $push (call $clip (call $pop)))
)
{{end}}


{{- if .HasOp "pan" -}}
;;-------------------------------------------------------------------------------
;;   PAN opcode: pan the signal
;;-------------------------------------------------------------------------------
;;   Mono:   s   ->  s*(1-p) s*p
;;   Stereo: l r ->  l*(1-p) r*p
;;
;;   where p is the panning in [0,1] range
;;-------------------------------------------------------------------------------
(func $su_op_pan (param $stereo i32)
{{- if .Stereo "pan"}}
    (if (i32.eqz (local.get $stereo)) (then ;; this time, if this is mono op...
        call $peek                         ;;    ...we duplicate the mono into stereo first
        call $push
    ))
    (call $pop)  ;; F: r        P: l
    (call $pop)  ;; F:          P: r l
    (call $input (i32.const {{.InputNumber "pan" "panning"}})) ;; F:        P: p r l
    f32.mul      ;; F:          P: p*r l
    (call $push) ;; F: p*r      P: l
    f32.const 1
    (call $input (i32.const {{.InputNumber "pan" "panning"}})) ;; F: p*r       P: p 1 l
    f32.sub      ;; F: p*r      P: 1-p l
    f32.mul      ;; F: p*r      P: (1-p)*l
    (call $push) ;; F: (1-p)*l p*r
{{- else}}
    (call $peek) ;; F: s       P: s
    (f32.mul
        (call $input (i32.const {{.InputNumber "pan" "panning"}}))
        (call $pop)
    )            ;; F:         P: p*s s
    (call $push) ;; F: p*s     P: s
    (call $peek) ;; F: p*s     P: p*s s
    f32.sub      ;; F: p*s     P: s-p*s
    (call $push) ;; F: (1-p)*s p*s
{{- end}}
)
{{end}}


{{- if .HasOp "delay"}}
;;-------------------------------------------------------------------------------
;;   DELAY opcode: adds delay effect to the signal
;;-------------------------------------------------------------------------------
;;   Mono:   perform delay on ST0, using delaycount delaylines starting
;;           at delayindex from the delaytable
;;   Stereo: perform delay on ST1, using delaycount delaylines starting
;;           at delayindex + delaycount from the delaytable (so the right delays
;;           can be different)
;;-------------------------------------------------------------------------------
(func $su_op_delay (param $stereo i32) (local $delayIndex i32) (local $delayCount i32) (local $output f32) (local $s f32) (local $filtstate f32)
{{- if .Stereo "delay"}} (local $delayCountStash i32) {{- end}}
{{- if or (.SupportsModulation "delay" "delaytime") (.SupportsParamValue "delay" "notetracking" 1)}} (local $delayTime f32) {{- end}}
    (local.set $delayIndex (i32.mul (call $scanOperand) (i32.const 2)))
{{- if .Stereo "delay"}}
    (local.set $delayCountStash (call $scanOperand))
    (if (local.get $stereo)(then
        (call $su_op_xch (i32.const 0))
    ))
    loop $stereoLoop
    (local.set $delayCount (local.get $delayCountStash))
{{- else}}
    (local.set $delayCount (call $scanOperand))
{{- end}}
    (local.set $output (f32.mul
        (call $input (i32.const {{.InputNumber "delay" "dry"}}))
        (call $peek)
    ))
    loop $delayLoop
        (local.tee $s (f32.load offset=12
            (i32.add ;; delayWRK + ((globalTick-delaytimes[delayIndex])&65535)*4
                (i32.mul ;; ((globalTick-delaytimes[delayIndex])&65535)*4
                    (i32.and ;; (globalTick-delaytimes[delayIndex])&65535
                        (i32.sub ;; globalTick-delaytimes[delayIndex]
                            (global.get $globaltick)
{{- if or (.SupportsModulation "delay" "delaytime") (.SupportsParamValue "delay" "notetracking" 1)}} ;; delaytime modulation or note syncing require computing the delay time in floats
{{- if .SupportsModulation "delay" "delaytime"}}
                            (local.set $delayTime (f32.add
                                (f32.convert_i32_u (i32.load16_u
                                    offset={{index .Labels "su_delay_times"}}
                                    (local.get $delayIndex)
                                ))
                                (f32.mul
                                    (f32.load offset={{.InputNumber "delay" "delaytime" | mul 4 | add 32}} (global.get $WRK))
                                    (f32.const 32767)
                                )
                            ))
{{- else}}
                            (local.set $delayTime (f32.convert_i32_u (i32.load16_u
                                    offset={{index .Labels "su_delay_times"}}
                                    (local.get $delayIndex)
                            )))
{{- end}}
{{- if .SupportsParamValue "delay" "notetracking" 1}}
                            (if (i32.eqz (i32.and (local.get $delayCount) (i32.const 1)))(then
                                (local.set $delayTime (f32.div
                                    (local.get $delayTime)
                                    (call $pow2
                                        (f32.mul
                                            (f32.convert_i32_u (i32.load (global.get $voice)))
                                            (f32.const 0.08333333)
                                        )
                                    )
                                ))
                            ))
{{- end}}
                            (i32.trunc_f32_s (f32.add (local.get $delayTime) (f32.const 0.5)))
{{- else}}
                            (i32.load16_u
                                offset={{index .Labels "su_delay_times"}}
                                (local.get $delayIndex)
                            )
{{- end}}
                        )
                        (i32.const 65535)
                    )
                    (i32.const 4)
                )
                (global.get $delayWRK)
            )
        ))
        (local.set $output (f32.add (local.get $output)))
        (f32.store
            (global.get $delayWRK)
            (local.tee $filtstate
                (f32.add
                    (f32.mul
                        (f32.sub
                            (f32.load (global.get $delayWRK))
                            (local.get $s)
                        )
                        (call $input (i32.const {{.InputNumber "delay" "damp"}}))
                    )
                    (local.get $s)
                )
            )
        )
        (f32.store offset=12
            (i32.add ;; delayWRK + globalTick*4
                (i32.mul ;; globalTick)&65535)*4
                    (i32.and ;; globalTick&65535
                        (global.get $globaltick)
                        (i32.const 65535)
                    )
                    (i32.const 4)
                )
                (global.get $delayWRK)
            )
            (f32.add
                (f32.mul
                    (call $input (i32.const {{.InputNumber "delay" "feedback"}}))
                    (local.get $filtstate)
                )
                (f32.mul
                    (f32.mul
                        (call $input (i32.const {{.InputNumber "delay" "pregain"}}))
                        (call $input (i32.const {{.InputNumber "delay" "pregain"}}))
                    )
                    (call $peek)
                )
            )
        )
        (global.set $delayWRK (i32.add (global.get $delayWRK) (i32.const 262156)))
        (local.set $delayIndex (i32.add (local.get $delayIndex) (i32.const 2)))
        (br_if $delayLoop (i32.gt_s (local.tee $delayCount (i32.sub (local.get $delayCount) (i32.const 2))) (i32.const 0)))
    end
    (f32.store offset=4
        (global.get $delayWRK)
        (local.tee $filtstate
            (f32.add
                (local.get $output)
                (f32.sub
                    (f32.mul
                        (f32.const 0.99609375)
                        (f32.load offset=4 (global.get $delayWRK))
                    )
                    (f32.load offset=8 (global.get $delayWRK))
                )
            )
        )
    )
    (f32.store offset=8
        (global.get $delayWRK)
        (local.get $output)
    )
    (drop (call $pop))
    (call $push (local.get $filtstate))
{{- if .Stereo "delay"}}
    (call $su_op_xch (i32.const 0))
    (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
    (call $su_op_xch (i32.const 0))
{{- end}}
{{- if .SupportsModulation "delay" "delaytime"}}
    (f32.store offset={{.InputNumber "delay" "delaytime" | mul 4 | add 32}} (global.get $WRK) (f32.const 0))
{{- end}}
)
{{end}}


{{- if .HasOp "compressor"}}
;;-------------------------------------------------------------------------------
;;   COMPRES opcode: push compressor gain to stack
;;-------------------------------------------------------------------------------
;;   Mono:   push g on stack, where g is a suitable gain for the signal
;;           you can either MULP to compress the signal or SEND it to a GAIN
;;           somewhere else for compressor side-chaining.
;;   Stereo: push g g on stack, where g is calculated using l^2 + r^2
;;-------------------------------------------------------------------------------
(func $su_op_compressor (param $stereo i32) (local $x2 f32) (local $level f32) (local $t2 f32)
{{- if .Stereo "compressor"}}
    (local.set $x2 (f32.mul
        (call $peek)
        (call $peek)
    ))
    (if (local.get $stereo)(then
        (call $pop)
        (local.set $x2 (f32.add
            (local.get $x2)
            (f32.mul
                (call $peek)
                (call $peek)
            )
        ))
        (call $push)
    ))
    (local.get $x2)
{{- else}}
    (local.tee $x2 (f32.mul
        (call $peek)
        (call $peek)
    ))
{{- end}}
    (local.tee $level (f32.load (global.get $WRK)))
    f32.lt
    call $nonLinearMap ;; $nonlinearMap(x^2<level) (let's call it c)
    (local.tee $level (f32.add ;; l'=l + c*(x^2-l)
        (f32.mul ;; c was already on stack, so c*(x^2-l)
            (f32.sub ;; x^2-l
                (local.get $x2)
                (local.get $level)
            )
        )
        (local.get $level)
    ))
    (local.tee $t2 (f32.mul ;; t^2
        (call $input (i32.const {{.InputNumber "compressor" "threshold"}}))
        (call $input (i32.const {{.InputNumber "compressor" "threshold"}}))
    ))
    (if (f32.gt) (then ;; if $level > $threshold, note the local.tees
        (call $push
            (call $pow ;; (t^2/l)^(r/2)
                (f32.div ;; t^2/l
                    (local.get $t2)
                    (local.get $level)
                )
                (f32.mul ;; r/2
                    (call $input (i32.const {{.InputNumber "compressor" "ratio"}})) ;; r
                    (f32.const 0.5)  ;; 0.5
                )
            )
        )
    )(else
        (call $push (f32.const 1)) ;; unity gain if we are below threshold
    ))
    (call $push (f32.div ;; apply post-gain ("make up gain")
        (call $pop)
        (call $input (i32.const {{.InputNumber "compressor" "invgain"}}))
    ))
{{- if .Stereo "compressor"}}
    (if (local.get $stereo)(then
        (call $push (call $peek))
    ))
{{- end}}
    (f32.store (global.get $WRK) (local.get $level)) ;; save the updated levels
)
{{- end}}
