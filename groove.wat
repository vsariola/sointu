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
(memory (export "m") 261)

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
(global $sp (mut i32) (i32.const 1408))
(global $outputBufPtr (mut i32) (i32.const 132736))
;; TODO: only export start and length with certain compiler options; in demo use, they can be hard coded
;; in the intro
(global $outputStart (export "s") i32 (i32.const 132736))
(global $outputLength (export "l") i32 (i32.const 16932864))
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
                (global.set $COM (i32.const 736))
                (global.set $VAL (i32.const 802))
                (global.set $WRK (i32.const 1600))
                (global.set $voice (i32.const 1600))
                (global.set $voicesRemain (i32.const 10))
                (call $su_run_vm)
            (i64.store (global.get $outputBufPtr) (i64.load (i32.const 1568))) ;; load the sample from left & right channels as one 64bit int and store it in the address pointed by outputBufPtr
            (global.set $outputBufPtr (i32.add (global.get $outputBufPtr) (i32.const 8)))      ;; advance outputbufptr
            (i64.store (i32.const 1568) (i64.const 0)) ;; clear the left and right ports
                (global.set $sample (i32.add (global.get $sample) (i32.const 1)))
                (global.set $globaltick (i32.add (global.get $globaltick) (i32.const 1)))
                (br_if $sample_loop (i32.lt_s (global.get $sample) (i32.const 5512)))
            end
            (global.set $row (i32.add (global.get $row) (i32.const 1)))
            (br_if $row_loop (i32.lt_s (global.get $row) (i32.const 16)))
        end
        (global.set $pattern (i32.add (global.get $pattern) (i32.const 1)))
        (br_if $pattern_loop (i32.lt_s (global.get $pattern) (i32.const 24)))
    end
)
;; the simple implementation of update_voices: each track has exactly one voice
(func $su_update_voices (local $si i32) (local $di i32) (local $tracksRemaining i32) (local $note i32)
    (local.set $tracksRemaining (i32.const 10))
    (local.set $si (global.get $pattern))
    (local.set $di (i32.const 1600))
    loop $track_loop
        (i32.load8_u offset=496 (local.get $si))
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
        (local.set $si (i32.add (local.get $si) (i32.const 24)))
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
                (if (i32.lt_u (local.get $paramNum) (i32.load8_u offset=1107 (local.get $opcode)))(then ;;(i32.ge (local.get $paramNum) (i32.load8_u (local.get $opcode)))  /*TODO: offset to transformvalues
                    (local.set $WRKplusparam (i32.add (global.get $WRK) (local.get $paramX4)))
                    (f32.store offset=1408
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
    (if (local.get $stereo)(then
        (call $push (call $peek))
    ))
)

;;-------------------------------------------------------------------------------
;;   OSCILLAT opcode: oscillator, the heart of the synth
;;-------------------------------------------------------------------------------
;;   Mono:   push oscillator value on stack
;;   Stereo: push l r on stack, where l has opposite detune compared to r
;;-------------------------------------------------------------------------------
(func $su_op_oscillator (param $stereo i32) (local $flags i32) (local $detune f32) (local $phase f32) (local $color f32) (local $amplitude f32)
    (local.set $flags (call $scanValueByte))
    (local.set $detune (call $inputSigned (i32.const 1)))
    loop $stereoLoop
    (f32.store ;; update phase
        (global.get $WRK)
        (local.tee $phase
            (f32.sub
                (local.tee $phase
                    ;; Transpose calculation starts
                    (f32.div
                        (call $inputSigned (i32.const 0))
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
                    (f32.add (f32.load (global.get $WRK))) ;; add the current phase of the oscillator
                )
                (f32.floor (local.get $phase))
            )
        )
    )
    (f32.add (local.get $phase) (call $input (i32.const 2)))
    (local.set $phase (f32.sub (local.tee $phase) (f32.floor (local.get $phase)))) ;; phase = phase mod 1.0
    (local.set $color (call $input (i32.const 3)))
    (if (i32.and (local.get $flags) (i32.const 0x40)) (then
        (local.set $amplitude (call $oscillator_sine (local.get $phase) (local.get $color)))
    ))
    (call $waveshaper (local.get $amplitude) (call $input (i32.const 4)))
    (call $push (f32.mul
        (call $input (i32.const 5))
    ))
    (local.set $detune (f32.neg (local.get $detune))) ;; flip the detune for secon round
    (global.set $WRK (i32.add (global.get $WRK) (i32.const 4))) ;; WARNING: this is a bug. WRK should be nonvolatile, but we are changing it. It does not cause immediate problems but modulations will be off.
    (br_if $stereoLoop (i32.eqz (local.tee $stereo (i32.eqz (local.get $stereo)))))
    end
)
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




;;-------------------------------------------------------------------------------
;;   OUT opcode: outputs and pops the signal
;;-------------------------------------------------------------------------------
;;   Stereo: add ST0 to left out and ST1 to right out, then pop
;;-------------------------------------------------------------------------------
(func $su_op_out (param $stereo i32) (local $ptr i32)
    (local.set $ptr (i32.const 1568)) ;; synth.left
    (f32.store (local.get $ptr)
        (f32.add
            (f32.mul
                (call $pop)
                (call $input (i32.const 0))
            )
            (f32.load (local.get $ptr))
        )
    )
    (local.set $ptr (i32.const 1572)) ;; synth.right, note that ATM does not seem to support mono ocpode at all
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
    (f32.load offset=1408 (i32.mul (local.get $inputNumber) (i32.const 4)))
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
(table 4 funcref)
(elem (i32.const 1) ;; start the indices at 1, as 0 is reserved for advance
    $su_op_envelope
    $su_op_oscillator
    $su_op_out
)



;; All data is collected into a byte buffer and emitted at once
(data (i32.const 0) "\4f\01\2b\01\4f\01\2b\01\4f\01\2b\01\4f\01\2b\01\01\00\00\00\00\00\00\00\00\00\00\00\00\00\00\00\4a\01\48\01\00\00\43\01\01\00\45\01\01\00\3e\01\1f\01\00\2b\01\00\2b\01\1f\01\00\00\00\00\00\00\1d\01\00\29\01\00\29\01\1d\01\00\00\00\00\00\00\00\00\00\00\4f\01\00\00\00\00\00\00\4f\00\00\00\41\01\00\45\01\00\46\01\01\01\01\01\01\01\01\00\41\01\00\41\01\01\3c\01\01\01\01\01\01\01\01\00\4f\01\2b\01\4f\01\2b\01\28\01\2b\01\4f\01\2b\01\4f\01\2b\01\14\01\2b\01\4f\01\2b\01\4f\01\2b\01\40\00\00\00\40\00\00\00\40\00\00\00\40\00\00\00\3e\01\00\41\01\00\41\01\01\01\01\01\01\01\01\00\45\01\00\3c\01\01\40\01\01\01\01\01\01\01\01\00\45\01\00\3e\01\00\3e\01\01\01\01\01\01\01\01\00\3c\01\00\45\01\01\43\01\01\01\01\01\01\01\01\00\4a\01\48\01\00\00\43\01\01\00\45\01\01\01\3e\01\1f\01\00\2b\01\00\2b\01\1f\01\01\01\01\01\01\00\1a\01\00\26\01\00\26\01\1a\01\00\00\00\00\00\00\00\00\00\00\4f\01\00\00\00\00\00\32\4f\00\00\1e\1a\01\00\26\01\00\26\01\1a\01\01\01\01\01\01\00\00\00\39\01\3e\01\39\45\01\39\43\01\41\00\40\00\4d\01\48\4a\01\01\48\01\01\01\01\01\01\00\48\4a\00\00\4f\01\01\00\4f\01\01\01\01\01\4d\01\4a\00\4d\01\00\4f\01\01\4d\01\01\01\4c\01\4a\01\48\01\18\01\00\24\01\00\24\01\18\01\01\01\01\01\01\00\3e\01\39\01\3e\01\39\45\01\39\43\01\41\01\40\00\3e\01\01\01\00\00\00\00\00\00\00\00\00\00\00\43\45\01\01\01\48\01\45\01\01\01\01\00\00\00\00\00\18\01\00\24\01\00\24\01\18\01\00\00\00\00\00\00\1d\01\00\29\01\00\29\01\1d\01\01\01\01\01\01\00\00\00\00\00\42\01\00\00\00\00\00\00\4f\00\00\00\00\00\00\00\00\00\00\00\01\00\00\00\01\00\00\00\00\00\00\00\00\00\00\00\02\03\04\05\02\03\04\05\06\07\08\09\06\07\08\09\02\03\04\05\02\03\04\05\0a\0a\0a\0a\0a\0a\0a\0a\0b\0c\0b\0c\0b\0c\0b\0c\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0d\0e\0d\0e\0d\0e\0d\0e\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0f\10\0f\10\0f\10\0f\10\0a\0a\0a\0a\0a\0a\0a\0a\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\11\0a\0a\0a\0a\12\12\12\13\12\12\12\14\12\12\12\14\12\12\12\12\12\12\12\12\0a\0a\0a\0a\0a\0a\0a\0a\15\16\17\18\15\16\17\18\19\1a\1b\01\19\1a\1b\01\1c\1c\1c\1c\1c\1c\1c\1c\1c\1c\1d\1e\1c\1c\1d\1e\1c\1c\1c\1c\1c\1c\1c\1c\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\0a\03\05\05\05\05\07\00\03\05\05\05\07\00\03\05\05\05\05\07\00\03\05\05\05\05\07\00\03\05\05\05\05\07\00\03\03\05\07\00\03\03\05\05\05\05\07\00\03\05\05\05\05\07\00\03\05\07\00\03\03\05\05\05\05\07\00\18\46\20\40\3c\00\00\00\00\00\40\40\00\00\00\00\00\80\40\00\00\00\00\00\80\40\00\00\00\00\00\30\40\41\20\46\3c\4b\20\00\00\00\00\00\80\40\00\00\00\00\00\80\40\00\00\00\00\00\80\40\12\20\46\50\46\50\00\00\00\00\00\78\40\00\00\00\00\00\80\40\00\00\00\00\00\40\40\00\00\00\00\00\08\40\20\20\46\50\46\50\00\00\00\00\00\78\40\00\00\00\00\00\80\40\00\00\00\00\00\40\40\00\00\00\00\00\08\40\20\20\46\50\46\50\00\00\00\00\00\78\40\00\00\00\00\00\80\40\00\00\00\00\00\40\40\00\00\00\00\00\08\40\20\00\40\60\40\5a\00\46\00\00\64\00\00\00\00\00\30\40\0c\00\48\00\48\19\00\38\00\00\80\00\00\00\00\00\40\40\00\00\00\00\00\40\40\00\00\00\00\00\40\40\00\00\00\00\00\10\40\04\20\40\5a\30\23\00\00\00\00\00\80\40\00\00\00\00\00\80\40\00\00\00\00\00\80\40\00\00\00\00\00\32\40\0a\00\40\0f\20\64\00\00\00\00\00\80\40\2c\00\46\00\46\40\00\50\00\50\80\00\00\00\00\00\80\40\00\00\00\00\00\80\40\00\00\00\00\00\80\40\00\00\00\00\00\80\40\10\05\06\01")

;;(data (i32.const 8388610) "\52\49\46\46\b2\eb\0c\20\57\41\56\45\66\6d\74\20\12\20\20\20\03\20\02\20\44\ac\20\20\20\62\05\20\08\20\20\20\20\20\66\61\63\74\04\20\20\20\e0\3a\03\20\64\61\74\61\80\eb\0c\20")

) ;; END MODULE
