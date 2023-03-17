(module


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
(memory (export "m") 831)

;;------------------------------------------------------------------------------
;; Globals. Putting all with same initialization value should compress most
;;------------------------------------------------------------------------------
(global $WRK (mut i32) (i32.const 0))
(global $COM (mut i32) (i32.const 0))
(global $VAL (mut i32) (i32.const 0))
(global $globaltick (mut i32) (i32.const 0))
(global $row (mut i32) (i32.const 0))
(global $pattern (mut i32) (i32.const 0))
(global $sample (mut i32) (i32.const 0))
(global $voice (mut i32) (i32.const 0))
(global $voicesRemain (mut i32) (i32.const 0))
(global $randseed (mut i32) (i32.const 1))
(global $sp (mut i32) (i32.const 2688))
(global $outputBufPtr (mut i32) (i32.const 134016))
;; TODO: only export start and length with certain compiler options; in demo use, they can be hard coded
;; in the intro
(global $outputStart (export "s") i32 (i32.const 134016))
(global $outputLength (export "l") i32 (i32.const 54326272))
(global $output16bit (export "t") i32 (i32.const 0))


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
    loop $pattern_loop
        (global.set $row (i32.const 0))
        loop $row_loop
            (call $su_update_voices)
            (global.set $sample (i32.const 0))
            loop $sample_loop
                (global.set $COM (i32.const 2210))
                (global.set $VAL (i32.const 2243))
                (global.set $WRK (i32.const 2880))
                (global.set $voice (i32.const 2880))
                (global.set $voicesRemain (i32.const 10))
                (call $su_run_vm)
            (i64.store (global.get $outputBufPtr) (i64.load (i32.const 2848))) ;; load the sample from left & right channels as one 64bit int and store it in the address pointed by outputBufPtr
            (global.set $outputBufPtr (i32.add (global.get $outputBufPtr) (i32.const 8)))      ;; advance outputbufptr
            (i64.store (i32.const 2848) (i64.const 0)) ;; clear the left and right ports
                (global.set $sample (i32.add (global.get $sample) (i32.const 1)))
                (global.set $globaltick (i32.add (global.get $globaltick) (i32.const 1)))
                (br_if $sample_loop (i32.lt_s (global.get $sample) (i32.const 5512)))
            end
            (global.set $row (i32.add (global.get $row) (i32.const 1)))
            (br_if $row_loop (i32.lt_s (global.get $row) (i32.const 16)))
        end
        (global.set $pattern (i32.add (global.get $pattern) (i32.const 1)))
        (br_if $pattern_loop (i32.lt_s (global.get $pattern) (i32.const 77)))
    end
)
;; the simple implementation of update_voices: each track has exactly one voice
(func $su_update_voices (local $si i32) (local $di i32) (local $tracksRemaining i32) (local $note i32)
    (local.set $tracksRemaining (i32.const 10))
    (local.set $si (global.get $pattern))
    (local.set $di (i32.const 2880))
    loop $track_loop
        (i32.load8_u offset=1440 (local.get $si))
        (i32.mul (i32.const 16))
        (i32.add (global.get $row))
        (i32.load8_u offset=0)
        (local.tee $note)
        (if (i32.ne (i32.const 1))(then
            (i32.store offset=4 (local.get $di) (i32.const 1)) ;; release the note
            (if (i32.gt_u (local.get $note) (i32.const 1))(then
                (memory.fill (local.get $di) (i32.const 0) (i32.const 4096))
                (i32.store (local.get $di) (local.get $note))
            ))
        ))
        (local.set $di (i32.add (local.get $di) (i32.const 4096)))
        (local.set $si (i32.add (local.get $si) (i32.const 77)))
        (br_if $track_loop (local.tee $tracksRemaining (i32.sub (local.get $tracksRemaining) (i32.const 1))))
    end
)

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
                (if (i32.lt_u (local.get $paramNum) (i32.load8_u offset=2317 (local.get $opcode)))(then ;;(i32.ge (local.get $paramNum) (i32.load8_u (local.get $opcode)))  /*TODO: offset to transformvalues
                    (local.set $WRKplusparam (i32.add (global.get $WRK) (local.get $paramX4)))
                    (f32.store offset=2688
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
            (br_if 2 (i32.eqz (global.get $voicesRemain))) ;; if no more voices remain, return from function
        ))
        br $vm_loop
    end
)




;;-------------------------------------------------------------------------------
;;   ENVELOPE opcode: pushes an ADSR envelope value on stack [0,1]
;;-------------------------------------------------------------------------------
;;   Mono:   push the envelope value on stack
;;   Stereo: push the envelope valeu on stack twice
;;-------------------------------------------------------------------------------
(func $su_op_envelope (param $stereo i32) (local $state i32) (local $level f32) (local $delta f32)
    (if (i32.load offset=4 (global.get $voice)) (then ;; if voice.release > 0
        (i32.store (global.get $WRK) (i32.const 3)) ;; set envelope state to release
    ))
    (local.set $state (i32.load (global.get $WRK)))
    (local.set $level (f32.load offset=4 (global.get $WRK)))
    (local.set $delta (call $nonLinearMap (local.get $state)))
    (if (local.get $state) (then
        (if (i32.eq (local.get $state) (i32.const 1))(then ;; state is 1 aka decay
            (local.set $level (f32.sub (local.get $level) (local.get $delta)))
            (if (f32.le (local.get $level) (call $input (i32.const 2)))(then
                (local.set $level (call $input (i32.const 2)))
                (local.set $state (i32.const 2))
            ))
        ))
        (if (i32.eq (local.get $state) (i32.const 3))(then ;; state is 3 aka release
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
    (call $push (f32.mul (local.get $level) (call $input (i32.const 4))))
)


;;-------------------------------------------------------------------------------
;;   OUT opcode: outputs and pops the signal
;;-------------------------------------------------------------------------------
;;   Mono: add ST0 to main left port, then pop
;;-------------------------------------------------------------------------------
(func $su_op_out (param $stereo i32) (local $ptr i32)
    (local.set $ptr (i32.const 2848)) ;; synth.left
    (f32.store (local.get $ptr)
        (f32.add
            (f32.mul
                (call $pop)
                (call $input (i32.const 0))
            )
            (f32.load (local.get $ptr))
        )
    )
    (local.set $ptr (i32.const 2852)) ;; synth.right, note that ATM does not seem to support mono ocpode at all
    (f32.store (local.get $ptr)
        (f32.add
            (f32.mul
                (call $pop)
                (call $input (i32.const 0))
            )
            (f32.load (local.get $ptr))
        )
    )
)



;;-------------------------------------------------------------------------------
;; $input returns the float value of a transformed to 0.0 - 1.0f range.
;; The transformed values start at 512 (TODO: change magic constants somehow)
;;-------------------------------------------------------------------------------
(func $input (param $inputNumber i32) (result f32)
    (f32.load offset=2688 (i32.mul (local.get $inputNumber) (i32.const 4)))
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
(table 3 funcref)
(elem (i32.const 1) ;; start the indices at 1, as 0 is reserved for advance
    $su_op_envelope
    $su_op_out
)



;; All data is collected into a byte buffer and emitted at once
(data (i32.const 0) "\4f\01\2b\01\4f\01\2b\01\4f\01\2b\01\4f\01\2b\01\01\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\4f\01\00\00\00\00\00\1e\4f\00\00\14\40\00\00\00\40\00\00\00\40\00\00\00\40\00\00\00\4a\01\48\01\00\00\43\01\01\00\45\01\01\00\3e\01\1f\01\00\2b\01\00\2b\01\1f\01\00\00\00\00\00\00\1d\01\00\29\01\00\29\01\1d\01\00\00\00\00\00\00\00\00\00\00\4f\01\00\00\00\00\00\00\4f\00\00\00\41\01\00\45\01\00\46\01\01\01\01\01\01\01\01\00\41\01\00\41\01\01\3c\01\01\01\01\01\01\01\01\00\4f\01\2b\01\4f\01\2b\01\28\01\2b\01\4f\01\2b\01\4f\01\2b\01\14\01\2b\01\4f\01\2b\01\4f\01\2b\01\00\00\00\00\00\00\00\00\43\01\01\01\45\01\01\01\43\01\00\00\00\00\00\43\00\43\01\00\00\00\00\00\45\01\00\00\00\00\00\45\00\45\01\00\00\00\00\00\43\00\00\43\00\00\00\00\43\00\00\43\00\00\00\00\40\01\00\00\00\00\00\40\00\00\00\00\00\00\00\00\37\01\01\01\01\01\01\01\35\01\01\01\01\01\01\01\40\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\3e\01\00\41\01\00\41\01\01\01\01\01\01\01\01\00\45\01\00\3c\01\01\40\01\01\01\01\01\01\01\01\00\48\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\46\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\45\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\41\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\00\00\00\00\00\00\00\00\40\01\01\01\41\01\01\01\46\01\00\00\00\00\00\46\00\46\01\00\00\00\00\00\41\01\00\00\00\00\00\41\00\41\01\00\00\00\00\00\40\00\00\40\00\00\00\00\3f\00\00\3f\00\00\00\00\3a\00\00\00\00\00\3a\01\01\01\00\00\00\00\00\00\3c\01\00\00\00\00\00\3c\00\00\00\00\00\00\00\00\3c\01\01\01\01\01\01\01\3a\01\01\01\01\01\01\01\45\01\00\3e\01\00\3e\01\01\01\01\01\01\01\01\00\3c\01\00\45\01\01\43\01\01\01\01\01\01\01\01\00\43\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\40\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\00\00\00\00\00\00\00\00\3c\01\01\01\3e\01\01\01\3e\01\00\00\00\00\00\3e\00\3e\01\00\00\00\00\00\3c\01\00\00\00\00\00\3c\00\3c\01\00\00\00\00\00\3c\00\00\3c\00\00\00\00\3c\00\00\3c\00\00\00\00\32\00\00\00\00\00\32\01\01\01\00\00\00\00\00\00\41\01\00\00\00\00\00\41\00\00\00\00\00\00\00\00\3f\01\01\01\01\01\01\01\3e\01\01\01\01\01\01\01\4a\01\48\01\00\00\43\01\01\00\45\01\01\01\3e\01\1f\01\00\2b\01\00\2b\01\1f\01\01\01\01\01\01\00\3c\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\00\00\00\00\00\00\00\00\00\00\3c\3e\41\01\43\00\4f\01\2b\14\4f\01\2b\01\4f\01\2b\14\4f\01\2b\1e\00\00\43\00\43\00\41\43\00\43\3e\00\41\00\43\00\21\01\01\2d\00\00\21\00\24\01\01\30\00\00\24\00\41\00\00\00\00\00\41\01\01\01\00\00\00\00\00\00\00\00\00\00\00\00\00\00\3c\3e\00\41\00\43\00\41\00\00\45\00\45\00\41\43\00\43\41\00\43\00\45\43\24\01\01\30\00\00\24\01\1f\01\01\2b\00\00\1f\01\00\00\00\00\4f\00\00\50\00\14\46\00\14\46\00\1e\1a\01\00\26\01\00\26\01\1a\01\00\00\00\00\00\00\18\01\00\24\01\00\24\01\1a\01\26\00\1d\00\29\01\50\0a\50\28\50\0a\50\1e\50\0a\50\28\50\28\50\1e\00\00\00\00\4f\01\00\00\00\00\00\32\4f\00\00\1e\1a\01\00\26\01\00\26\01\1a\01\01\01\01\01\01\00\00\00\39\01\3e\01\39\45\01\39\43\01\41\00\40\00\40\00\00\00\00\00\00\00\00\14\40\00\00\00\00\00\1d\01\01\01\00\00\00\29\00\29\1d\00\24\00\29\00\41\3e\00\00\00\00\43\00\00\00\45\00\00\00\00\00\4d\01\48\4a\01\01\48\01\01\01\01\01\01\00\48\4a\00\00\4f\01\01\00\4f\01\01\01\01\01\4d\01\4a\00\4d\01\00\4f\01\01\4d\01\01\01\4c\01\4a\01\48\01\18\01\00\24\01\00\24\01\18\01\01\01\01\01\01\00\3e\01\39\01\3e\01\39\45\01\39\43\01\41\01\40\00\3e\01\01\01\00\00\00\00\00\00\00\00\00\00\00\43\45\01\01\01\48\01\45\01\01\01\01\00\00\00\00\00\00\00\00\00\00\00\00\45\4a\00\4a\00\45\01\4a\4c\00\00\00\00\00\00\00\45\4c\00\4c\01\45\4c\00\4c\4d\01\4c\01\4a\01\48\45\01\00\00\00\00\00\00\43\45\01\48\01\45\01\43\45\41\01\01\01\00\00\00\00\3e\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\40\00\00\00\40\00\00\00\40\00\00\20\40\00\00\00\18\01\00\24\01\00\24\01\18\01\00\00\00\00\00\00\1d\01\00\29\01\00\29\01\1d\01\01\01\01\01\01\00\00\00\00\00\42\01\00\00\00\00\00\00\4f\00\00\00\4d\01\4c\01\00\00\48\01\01\00\43\01\01\40\01\00\1f\01\01\01\00\00\00\2b\00\2b\1f\00\26\00\2b\00\45\00\00\00\00\00\00\00\45\48\45\3c\43\00\41\00\48\43\00\43\00\00\41\43\00\00\45\43\00\00\41\3e\00\00\00\00\4f\00\00\50\00\1e\00\00\50\00\00\00\39\01\00\00\00\00\00\39\00\00\00\00\00\00\00\00\00\43\41\43\48\00\41\43\00\43\41\43\48\00\43\41\21\01\01\01\00\00\00\2d\00\2d\21\00\28\00\2d\00\41\3e\00\00\00\00\43\00\00\00\45\00\48\00\4a\00\1a\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\00\00\00\00\00\00\00\00\01\00\00\00\01\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\02\03\03\03\03\03\03\03\03\03\03\03\03\03\03\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\03\03\03\04\05\06\07\04\05\06\07\08\09\0a\0b\08\09\0a\0b\04\05\06\07\04\05\06\07\04\05\06\07\04\05\06\07\05\05\05\0c\0d\0e\0f\0d\0d\0e\0f\0d\0e\10\11\0e\10\11\04\04\04\04\04\05\06\07\04\05\06\07\04\05\06\07\04\05\06\07\04\04\04\12\03\03\03\03\03\03\03\03\03\03\03\13\14\13\14\13\14\13\14\03\03\03\03\03\03\03\03\15\16\17\18\15\16\17\18\03\03\03\19\1a\1b\1c\1d\1a\1b\1c\1d\1e\1e\1f\1e\1e\1f\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\20\21\20\21\20\21\20\21\03\03\03\03\03\03\03\03\22\23\23\22\22\23\23\22\03\03\03\24\25\26\27\28\25\26\27\28\29\29\2a\29\29\2a\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\2b\2c\2b\2c\2b\2c\2b\2c\03\03\03\03\03\03\03\03\18\2d\15\2e\18\2d\15\2e\03\03\03\2f\30\31\32\33\30\31\32\33\34\35\36\34\35\36\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\38\39\39\39\39\39\39\39\39\39\39\39\39\39\39\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\37\01\03\03\03\03\03\03\03\3a\3a\3a\3b\3a\3a\3a\3c\3a\3a\3a\3c\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3d\3e\3e\3e\3e\3e\3e\3e\3e\3e\3e\3f\3e\3e\3f\03\03\03\03\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\3a\03\03\03\03\03\03\03\03\03\03\03\03\40\41\42\43\40\41\42\43\44\45\46\47\44\45\46\47\48\49\4a\4b\48\49\4a\4b\4c\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\40\41\42\43\40\41\42\43\03\03\03\03\03\03\03\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4e\4f\4d\4d\4e\4f\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\50\51\51\51\51\51\51\51\51\51\51\51\51\51\51\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\4d\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\03\52\53\54\03\52\53\54\55\56\57\55\56\57\55\58\58\58\58\58\58\58\58\59\59\59\59\59\59\59\59\59\59\59\59\59\59\59\59\03\03\03\02\04\00\02\04\00\02\04\00\02\04\00\02\04\00\02\02\04\00\02\02\04\00\02\04\00\02\04\00\02\02\04\00\18\46\20\40\3c\41\20\46\3c\4b\20\12\20\46\50\46\50\20\20\46\50\46\50\20\20\46\50\46\50\20\00\40\60\40\5a\00\46\00\00\64\0c\00\48\00\48\19\00\38\00\00\80\04\20\40\5a\30\23\0a\00\40\0f\20\64\2c\00\46\00\46\40\00\50\00\50\80\10\05\01")

;;(data (i32.const 8388610) "\52\49\46\46\b2\eb\0c\20\57\41\56\45\66\6d\74\20\12\20\20\20\03\20\02\20\44\ac\20\20\20\62\05\20\08\20\20\20\20\20\66\61\63\74\04\20\20\20\e0\3a\03\20\64\61\74\61\80\eb\0c\20")

) ;; END MODULE
