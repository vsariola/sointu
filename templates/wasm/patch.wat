;;-------------------------------------------------------------------------------
;;   su_run_vm function: runs the entire virtual machine once, creating 1 sample
;;-------------------------------------------------------------------------------
(func $su_run_vm (local $opcodeWithStereo i32) (local $opcode i32) (local $paramNum i32) (local $paramX4 i32) (local $WRKplusparam i32)
    loop $vm_loop
        (local.set $opcodeWithStereo (i32.load8_u (global.get $COM)))
        (global.set $COM (i32.add (global.get $COM) (i32.const 1)))  ;; move to next instruction
        (global.set $WRK (i32.add (global.get $WRK) (i32.const 64))) ;; move WRK to next unit
        (if (local.tee $opcode (i32.shr_u (local.get $opcodeWithStereo) (i32.const 1)))(then ;; if $opcode = $opcodeStereo >> 1; $opcode != 0 {
            (local.set $paramNum (i32.const 0))
            (local.set $paramX4 (i32.const 0))
            loop $transform_values_loop
                {{- $addr := sub (index .Labels "su_vm_transformcounts") 1}}
                (if (i32.lt_u (local.get $paramNum) (i32.load8_u offset={{$addr}} (local.get $opcode)))(then ;;(i32.ge (local.get $paramNum) (i32.load8_u (local.get $opcode)))  /*TODO: offset to transformvalues
                    (local.set $WRKplusparam (i32.add (global.get $WRK) (local.get $paramX4)))
                    (f32.store offset={{index .Labels "su_transformedvalues"}}
                        (local.get $paramX4)
                        (f32.add
                            (f32.mul
                                (f32.convert_i32_u (call $scanValueByte))
                                (f32.const 0.0078125) ;; scale from 0-128 to 0.0 - 1.0
                            )
                            (f32.load offset=32 (local.get $WRKplusparam)) ;; add modulation
                        )
                    )
                    (f32.store offset=32 (local.get $WRKplusparam) (f32.const 0.0)) ;; clear modulations
                    (local.set $paramNum (i32.add (local.get $paramNum) (i32.const 1))) ;; $paramNum++
                    (local.set $paramX4 (i32.add (local.get $paramX4) (i32.const 4)))
                    br $transform_values_loop ;; continue looping
                ))
                ;; paramNum was >= the number of parameters to transform, exiting loop
            end
            (call_indirect (type $opcode_func_signature) (i32.and (local.get $opcodeWithStereo) (i32.const 1)) (local.get $opcode))
        )(else ;; advance to next voice
            (global.set $voice (i32.add (global.get $voice) (i32.const 4096))) ;; advance to next voice
            (global.set $WRK (global.get $voice)) ;; set WRK point to beginning of voice
            (global.set $voicesRemain (i32.sub (global.get $voicesRemain) (i32.const 1)))
{{- if .SupportsPolyphony}}
            (if (i32.and (i32.shr_u (i32.const {{.PolyphonyBitmask | printf "%v"}}) (global.get $voicesRemain)) (i32.const 1))(then
                (global.set $VAL (global.get $VAL_instr_start))
                (global.set $COM (global.get $COM_instr_start))
            ))
            (global.set $VAL_instr_start (global.get $VAL))
            (global.set $COM_instr_start (global.get $COM))
{{- end}}
            (br_if 2 (i32.eqz (global.get $voicesRemain))) ;; if no more voices remain, return from function
        ))
        br $vm_loop
    end
)

{{- template "arithmetic.wat" .}}
{{- template "effects.wat" .}}
{{- template "sources.wat" .}}
{{- template "sinks.wat" .}}

;;-------------------------------------------------------------------------------
;; $input returns the float value of a transformed to 0.0 - 1.0f range.
;; The transformed values start at 512 (TODO: change magic constants somehow)
;;-------------------------------------------------------------------------------
(func $input (param $inputNumber i32) (result f32)
    (f32.load offset={{index .Labels "su_transformedvalues"}} (i32.mul (local.get $inputNumber) (i32.const 4)))
)

;;-------------------------------------------------------------------------------
;; $inputSigned returns the float value of a transformed to -1.0 - 1.0f range.
;;-------------------------------------------------------------------------------
(func $inputSigned (param $inputNumber i32) (result f32)
    (f32.sub (f32.mul (call $input (local.get $inputNumber)) (f32.const 2)) (f32.const 1))
)

;;-------------------------------------------------------------------------------
;; $nonLinearMap: x -> 2^(-24*input[x])
;;-------------------------------------------------------------------------------
(func $nonLinearMap (param $value i32) (result f32)
    (call $pow2
        (f32.mul
            (f32.const -24)
            (call $input (local.get $value))
        )
    )
)

;;-------------------------------------------------------------------------------
;; $pow2: x -> 2^x
;;-------------------------------------------------------------------------------
(func $pow2 (param $value f32) (result f32)
    (call $pow (f32.const 2) (local.get $value))
)

;;-------------------------------------------------------------------------------
;; Waveshaper(x,a): "distorts" signal x by amount a
;; Returns  x*a/(1-a+(2*a-1)*abs(x))
;;-------------------------------------------------------------------------------
(func $waveshaper (param $signal f32) (param $amount f32) (result f32)
    (local.set $signal (call $clip (local.get $signal)))
    (f32.mul
        (local.get $signal)
        (f32.div
            (local.get $amount)
            (f32.add
                (f32.const 1)
                (f32.sub
                    (f32.mul
                        (f32.sub
                            (f32.add (local.get $amount) (local.get $amount))
                            (f32.const 1)
                        )
                        (f32.abs (local.get $signal))
                    )
                    (local.get $amount)
                )
            )
        )
    )
)

;;-------------------------------------------------------------------------------
;; Clip(a : f32) returns min(max(a,-1),1)
;;-------------------------------------------------------------------------------
(func $clip (param $value f32) (result f32)
    (f32.min (f32.max (local.get $value) (f32.const -1.0)) (f32.const 1.0))
)

(func $stereoHelper (param $stereo i32) (param $tableIndex i32)
    (if (local.get $stereo)(then
        (call $pop)
        (global.set $WRK (i32.add (global.get $WRK) (i32.const 16)))
        (call_indirect (type $opcode_func_signature) (i32.const 0) (local.get $tableIndex))
        (global.set $WRK (i32.sub (global.get $WRK) (i32.const 16)))
        (call $push)
    ))
)

;;-------------------------------------------------------------------------------
;; The opcode table jump table. This is constructed to only include the opcodes
;; that are used so that the jump table is as small as possible.
;;-------------------------------------------------------------------------------
(table {{.Instructions | len | add 1}} anyfunc)
(elem (i32.const 1) ;; start the indices at 1, as 0 is reserved for advance
{{- range .Instructions}}
    $su_op_{{.}}
{{- end}}
)
