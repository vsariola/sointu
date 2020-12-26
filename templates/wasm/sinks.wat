{{- if .HasOp "outaux"}}
;;-------------------------------------------------------------------------------
;;   OUTAUX opcode: outputs to main and aux1 outputs and pops the signal
;;-------------------------------------------------------------------------------
;;   Mono: add outgain*ST0 to main left port and auxgain*ST0 to aux1 left
;;   Stereo: also add outgain*ST1 to main right port and auxgain*ST1 to aux1 right
;;-------------------------------------------------------------------------------
(func $su_op_outaux (param $stereo i32) (local $addr i32)
    (local.set $addr (i32.const 4128))
{{- if .Stereo "outaux"}}
    loop $stereoLoop
{{- end}}
        (f32.store ;; send
            (local.get $addr)
            (f32.add
                (f32.mul
                    (call $peek)
                    (call $input (i32.const {{.InputNumber "outaux" "outgain"}}))
                )
                (f32.load (local.get $addr))
            )
        )
        (f32.store offset=8
            (local.get $addr)
            (f32.add
                (f32.mul
                    (call $pop)
                    (call $input (i32.const {{.InputNumber "outaux" "auxgain"}}))
                )
                (f32.load offset=8 (local.get $addr))
            )
        )
{{- if .Stereo "outaux"}}
        (local.set $addr (i32.add (local.get $addr) (i32.const 4)))
        (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
{{- end}}
)
{{end}}


{{- if .HasOp "aux"}}
;;-------------------------------------------------------------------------------
;;   AUX opcode: outputs the signal to aux (or main) port and pops the signal
;;-------------------------------------------------------------------------------
;;   Mono: add gain*ST0 to left port
;;   Stereo: also add gain*ST1 to right port
;;-------------------------------------------------------------------------------
(func $su_op_aux (param $stereo i32) (local $addr i32)
    (local.set $addr (i32.add (i32.mul (call $scanValueByte) (i32.const 4)) (i32.const 4128)))
{{- if .Stereo "aux"}}
    loop $stereoLoop
{{- end}}
        (f32.store
            (local.get $addr)
            (f32.add
                (f32.mul
                    (call $pop)
                    (call $input (i32.const {{.InputNumber "aux" "gain"}}))
                )
                (f32.load (local.get $addr))
            )
        )
{{- if .Stereo "aux"}}
        (local.set $addr (i32.add (local.get $addr) (i32.const 4)))
        (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
{{- end}}
)
{{end}}


{{- if .HasOp "send"}}
;;-------------------------------------------------------------------------------
;;   SEND opcode: adds the signal to a port
;;-------------------------------------------------------------------------------
;;   Mono: adds signal to a memory address, defined by a word in VAL stream
;;   Stereo: also add right signal to the following address
;;-------------------------------------------------------------------------------
(func $su_op_send (param $stereo i32) (local $address i32) (local $scaledAddress i32)
    (local.set $address (i32.add (call $scanValueByte) (i32.shl (call $scanValueByte) (i32.const 8))))
    (if (i32.eqz (i32.and (local.get $address) (i32.const 8)))(then
{{- if .Stereo "send"}}
        (if (local.get $stereo)(then
            (call $push (call $peek2))
            (call $push (call $peek2))
        )(else
{{- end}}
            (call $push (call $peek))
{{- if .Stereo "send"}}
        ))
{{- end}}
    ))
{{- if .Stereo "send"}}
    loop $stereoLoop
{{- end}}
    (local.set $scaledAddress (i32.add (i32.mul (i32.and (local.get $address) (i32.const 0x7FF7)) (i32.const 4))
{{- if .SupportsParamValueOtherThan "send" "voice" 0}}
        (select
            (i32.const 4096)
{{- end}}
            (global.get $voice)
{{- if .SupportsParamValueOtherThan "send" "voice" 0}}
            (i32.and (local.get $address)(i32.const 0x8000))
        )
{{- end}}
    ))
    (f32.store offset=32
        (local.get $scaledAddress)
        (f32.add
            (f32.load offset=32 (local.get $scaledAddress))
            (f32.mul
                (call $inputSigned (i32.const {{.InputNumber "send" "amount"}}))
                (call $pop)
            )
        )
    )
    {{- if .Stereo "send"}}
    (local.set $address (i32.add (local.get $address) (i32.const 1)))
    (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
    {{- end}}
)
{{end}}



{{- if .HasOp "out"}}
;;-------------------------------------------------------------------------------
;;   OUT opcode: outputs and pops the signal
;;-------------------------------------------------------------------------------
{{- if .Mono "out"}}
;;   Mono: add ST0 to main left port, then pop
{{- end}}
{{- if .Stereo "out"}}
;;   Stereo: add ST0 to left out and ST1 to right out, then pop
{{- end}}
;;-------------------------------------------------------------------------------
(func $su_op_out (param $stereo i32) (local $ptr i32)
    (local.set $ptr (i32.const 4128)) ;; synth.left, but should not be magic constant
    (f32.store (local.get $ptr)
        (f32.add
            (f32.mul
                (call $pop)
                (call $input (i32.const {{.InputNumber "out" "gain"}}))
            )
            (f32.load (local.get $ptr))
        )
    )
    (local.set $ptr (i32.const 4132)) ;; synth.right, but should not be magic constant
    (f32.store (local.get $ptr)
        (f32.add
            (f32.mul
                (call $pop)
                (call $input (i32.const {{.InputNumber "out" "gain"}}))
            )
            (f32.load (local.get $ptr))
        )
    )
)
{{end}}

{{- if .HasOp "speed"}}
;;-------------------------------------------------------------------------------
;;   SPEED opcode: modulate the speed (bpm) of the song based on ST0
;;-------------------------------------------------------------------------------
;;   Mono: adds or subtracts the ticks, a value of 0.5 is neutral & will7
;;   result in no speed change.
;;   There is no STEREO version.
;;-------------------------------------------------------------------------------
(func $su_op_speed (param $stereo i32) (local $r f32) (local $w i32)
    (f32.store
        (global.get $WRK)
        (local.tee $r
            (f32.sub
                (local.tee $r
                    (f32.add
                        (f32.load (global.get $WRK))
                        (f32.sub
                            (call $pow2
                                (f32.mul
                                    (call $pop)
                                    (f32.const 2.206896551724138)
                                )
                            )
                            (f32.const 1)
                        )
                    )
                )
                (f32.convert_i32_s
                    (local.tee $w (i32.trunc_f32_s (local.get $r))) ;; note: small difference from x86, as this is trunc; x86 rounds to nearest)
                )
            )
        )
    )
    (global.set $sample (i32.add (global.get $sample) (local.get $w)))
)
{{end}}
