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
{{- .SetBlockLabel "su_hold_override"}}
{{- .Block 32}}
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

{{- /*
;-------------------------------------------------------------------------------
; Pow function ( initialized below )
;-------------------------------------------------------------------------------
*/}}
{{- .Align}}

{{- .SetBlockLabel "math_pow_1"}}
{{- .Block 262}}
{{- .SetBlockLabel "math_pow_2"}}
{{- .Block 240}}

;;------------------------------------------------------------------------------
;; Import the difficult math functions from javascript
;; (seriously now, it's 2020)
;;------------------------------------------------------------------------------

(func $log2 (param $0 f32) (result f32)
  (local $1 i32)
  (local $2 f64)
  (local $3 i32)
  (local $4 i32)
  (local $5 f64)
  block $~lib/util/math/log2f_lut|inlined.0 (result f32)
   local.get $0
   i32.reinterpret_f32
   local.tee $1
   i32.const 8388608
   i32.sub
   i32.const 2130706432
   i32.ge_u
   if
    f32.const -inf
    local.get $1
    i32.const 1
    i32.shl
    i32.eqz
    br_if $~lib/util/math/log2f_lut|inlined.0
    drop
    local.get $0
    local.get $1
    i32.const 2139095040
    i32.eq
    br_if $~lib/util/math/log2f_lut|inlined.0
    drop
    local.get $1
    i32.const 31
    i32.shr_u
    local.get $1
    i32.const 1
    i32.shl
    i32.const -16777216
    i32.ge_u
    i32.or
    if
     local.get $0
     local.get $0
     f32.sub
     local.tee $0
     local.get $0
     f32.div
     br $~lib/util/math/log2f_lut|inlined.0
    end
    local.get $0
    f32.const 8388608
    f32.mul
    i32.reinterpret_f32
    i32.const 192937984
    i32.sub
    local.set $1
   end
   local.get $1
   i32.const 1060306944
   i32.sub
   local.tee $3
   i32.const 19
   i32.shr_u
   i32.const 15
   i32.and
   i32.const 4
   i32.shl
   i32.const {{index .Labels "math_pow_1"}}
   i32.add
   local.set $4
   local.get $1
   local.get $3
   i32.const -8388608
   i32.and
   i32.sub
   f32.reinterpret_i32
   f64.promote_f32
   local.get $4
   f64.load
   f64.mul
   f64.const 1
   f64.sub
   local.tee $2
   local.get $2
   f64.mul
   local.set $5
   local.get $2
   f64.const 0.4811247078767291
   f64.mul
   f64.const -0.7213476299867769
   f64.add
   local.get $5
   f64.const -0.36051725506874704
   f64.mul
   f64.add
   local.get $5
   f64.mul
   local.get $2
   f64.const 1.4426950186867042
   f64.mul
   local.get $4
   f64.load offset=8
   local.get $3
   i32.const 23
   i32.shr_s
   f64.convert_i32_s
   f64.add
   f64.add
   f64.add
   f32.demote_f64
  end
)

(func $~lib/math/NativeMathf.pow (param $0 f32) (param $1 f32) (result f32)
  (local $2 i32)
  (local $3 f64)
  (local $4 i32)
  (local $5 i64)
  (local $6 i32)
  (local $7 i32)
  (local $8 f64)
  local.get $1
  f32.abs
  f32.const 2
  f32.le
  if
   local.get $1
   f32.const 2
   f32.eq
   if
    local.get $0
    local.get $0
    f32.mul
    return
   end
   local.get $1
   f32.const 0.5
   f32.eq
   if
    local.get $0
    f32.sqrt
    f32.abs
    f32.const inf
    local.get $0
    f32.const -inf
    f32.ne
    select
    return
   end
   local.get $1
   f32.const -1
   f32.eq
   if
    f32.const 1
    local.get $0
    f32.div
    return
   end
   local.get $1
   f32.const 1
   f32.eq
   if
    local.get $0
    return
   end
   local.get $1
   f32.const 0
   f32.eq
   if
    f32.const 1
    return
   end
  end
  block $~lib/util/math/powf_lut|inlined.0 (result f32)
   local.get $1
   i32.reinterpret_f32
   local.tee $7
   i32.const 1
   i32.shl
   i32.const 1
   i32.sub
   i32.const -16777217
   i32.ge_u
   local.tee $6
   local.get $0
   i32.reinterpret_f32
   local.tee $2
   i32.const 8388608
   i32.sub
   i32.const 2130706432
   i32.ge_u
   i32.or
   if
    local.get $6
    if
     f32.const 1
     local.get $7
     i32.const 1
     i32.shl
     i32.eqz
     br_if $~lib/util/math/powf_lut|inlined.0
     drop
     f32.const nan:0x400000
     local.get $2
     i32.const 1065353216
     i32.eq
     br_if $~lib/util/math/powf_lut|inlined.0
     drop
     local.get $0
     local.get $1
     f32.add
     local.get $7
     i32.const 1
     i32.shl
     i32.const -16777216
     i32.gt_u
     local.get $2
     i32.const 1
     i32.shl
     i32.const -16777216
     i32.gt_u
     i32.or
     br_if $~lib/util/math/powf_lut|inlined.0
     drop
     f32.const nan:0x400000
     local.get $2
     i32.const 1
     i32.shl
     i32.const 2130706432
     i32.eq
     br_if $~lib/util/math/powf_lut|inlined.0
     drop
     f32.const 0
     local.get $7
     i32.const 31
     i32.shr_u
     i32.eqz
     local.get $2
     i32.const 1
     i32.shl
     i32.const 2130706432
     i32.lt_u
     i32.eq
     br_if $~lib/util/math/powf_lut|inlined.0
     drop
     local.get $1
     local.get $1
     f32.mul
     br $~lib/util/math/powf_lut|inlined.0
    end
    local.get $2
    i32.const 1
    i32.shl
    i32.const 1
    i32.sub
    i32.const -16777217
    i32.ge_u
    if
     f32.const 1
     local.get $0
     local.get $0
     f32.mul
     local.tee $0
     f32.neg
     local.get $0
     local.get $2
     i32.const 31
     i32.shr_u
     if (result i32)
      block $~lib/util/math/checkintf|inlined.0 (result i32)
       i32.const 0
       local.get $7
       i32.const 23
       i32.shr_u
       i32.const 255
       i32.and
       local.tee $2
       i32.const 127
       i32.lt_u
       br_if $~lib/util/math/checkintf|inlined.0
       drop
       i32.const 2
       local.get $2
       i32.const 150
       i32.gt_u
       br_if $~lib/util/math/checkintf|inlined.0
       drop
       i32.const 0
       i32.const 1
       i32.const 150
       local.get $2
       i32.sub
       i32.shl
       local.tee $2
       i32.const 1
       i32.sub
       local.get $7
       i32.and
       br_if $~lib/util/math/checkintf|inlined.0
       drop
       i32.const 1
       local.get $2
       local.get $7
       i32.and
       br_if $~lib/util/math/checkintf|inlined.0
       drop
       i32.const 2
      end
      i32.const 1
      i32.eq
     else
      i32.const 0
     end
     select
     local.tee $0
     f32.div
     local.get $0
     local.get $7
     i32.const 31
     i32.shr_u
     select
     br $~lib/util/math/powf_lut|inlined.0
    end
    local.get $2
    i32.const 31
    i32.shr_u
    if
     block $~lib/util/math/checkintf|inlined.1 (result i32)
      i32.const 0
      local.get $7
      i32.const 23
      i32.shr_u
      i32.const 255
      i32.and
      local.tee $4
      i32.const 127
      i32.lt_u
      br_if $~lib/util/math/checkintf|inlined.1
      drop
      i32.const 2
      local.get $4
      i32.const 150
      i32.gt_u
      br_if $~lib/util/math/checkintf|inlined.1
      drop
      i32.const 0
      i32.const 1
      i32.const 150
      local.get $4
      i32.sub
      i32.shl
      local.tee $4
      i32.const 1
      i32.sub
      local.get $7
      i32.and
      br_if $~lib/util/math/checkintf|inlined.1
      drop
      i32.const 1
      local.get $4
      local.get $7
      i32.and
      br_if $~lib/util/math/checkintf|inlined.1
      drop
      i32.const 2
     end
     local.tee $4
     i32.eqz
     if
      local.get $0
      local.get $0
      f32.sub
      local.tee $0
      local.get $0
      f32.div
      br $~lib/util/math/powf_lut|inlined.0
     end
     i32.const 65536
     i32.const 0
     local.get $4
     i32.const 1
     i32.eq
     select
     local.set $4
     local.get $2
     i32.const 2147483647
     i32.and
     local.set $2
    end
    local.get $2
    i32.const 8388608
    i32.lt_u
    if (result i32)
     local.get $0
     f32.const 8388608
     f32.mul
     i32.reinterpret_f32
     i32.const 2147483647
     i32.and
     i32.const 192937984
     i32.sub
    else
     local.get $2
    end
    local.set $2
   end
   local.get $2
   local.get $2
   i32.const 1060306944
   i32.sub
   local.tee $2
   i32.const -8388608
   i32.and
   local.tee $6
   i32.sub
   f32.reinterpret_i32
   f64.promote_f32
   local.get $2
   i32.const 19
   i32.shr_u
   i32.const 15
   i32.and
   i32.const 4
   i32.shl
   i32.const {{index .Labels "math_pow_1"}}
   i32.add
   local.tee $2
   f64.load
   f64.mul
   f64.const 1
   f64.sub
   local.tee $3
   local.get $3
   f64.mul
   local.set $8
   local.get $1
   f64.promote_f32
   local.get $3
   f64.const 0.288457581109214
   f64.mul
   f64.const -0.36092606229713164
   f64.add
   local.get $8
   local.get $8
   f64.mul
   f64.mul
   local.get $3
   f64.const 1.4426950408774342
   f64.mul
   local.get $2
   f64.load offset=8
   local.get $6
   i32.const 23
   i32.shr_s
   f64.convert_i32_s
   f64.add
   f64.add
   local.get $3
   f64.const 0.480898481472577
   f64.mul
   f64.const -0.7213474675006291
   f64.add
   local.get $8
   f64.mul
   f64.add
   f64.add
   f64.mul
   local.tee $3
   i64.reinterpret_f64
   i64.const 47
   i64.shr_u
   i64.const 65535
   i64.and
   i64.const 32959
   i64.ge_u
   if
    f32.const -1584563250285286751870879e5
    f32.const 1584563250285286751870879e5
    local.get $4
    select
    f32.const 1584563250285286751870879e5
    f32.mul
    local.get $3
    f64.const 127.99999995700433
    f64.gt
    br_if $~lib/util/math/powf_lut|inlined.0
    drop
    f32.const -2.524354896707238e-29
    f32.const 2.524354896707238e-29
    local.get $4
    select
    f32.const 2.524354896707238e-29
    f32.mul
    local.get $3
    f64.const -150
    f64.le
    br_if $~lib/util/math/powf_lut|inlined.0
    drop
   end
   local.get $3
   f64.const 211106232532992
   f64.add
   local.tee $8
   i64.reinterpret_f64
   local.set $5
   local.get $3
   local.get $8
   f64.const 211106232532992
   f64.sub
   f64.sub
   local.tee $3
   f64.const 0.6931471806916203
   f64.mul
   f64.const 1
   f64.add
   local.get $3
   f64.const 0.05550361559341535
   f64.mul
   f64.const 0.2402284522445722
   f64.add
   local.get $3
   local.get $3
   f64.mul
   f64.mul
   f64.add
   local.get $5
   i32.wrap_i64
   i32.const 31
   i32.and
   i32.const 3
   i32.shl
   i32.const {{add (index .Labels "math_pow_1") 256}}
   i32.add
   i64.load
   local.get $4
   i64.extend_i32_u
   local.get $5
   i64.add
   i64.const 47
   i64.shl
   i64.add
   f64.reinterpret_i64
   f64.mul
   f32.demote_f64
  end
 )
 (func $pow (param $0 f32) (param $1 f32) (result f32)
  local.get $0
  local.get $1
  call $~lib/math/NativeMathf.pow
 )
 (export "pow" (func $pow))

(func $sin (param $0 f32) (result f32)
  (local $1 f32)
  local.get $0
  f32.const 0.31830987334251404
  f32.mul
  local.tee $0
  f32.floor
  local.set $1
  local.get $0
  local.get $1
  f32.sub
  local.tee $0
  f32.const 1
  local.get $0
  f32.sub
  f32.mul
  local.tee $0
  local.get $0
  f32.const 3.5999999046325684
  f32.mul
  f32.const 3.0999999046325684
  f32.add
  f32.mul
  local.tee $0
  f32.neg
  local.get $0
  local.get $1
  i32.trunc_sat_f32_s
  i32.const 1
  i32.and
  select
)

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
(global $globaltick (export "tick") (mut i32) (i32.const 0))
(global $row (export "row") (mut i32) (i32.const 0))
(global $pattern (export "pattern") (mut i32) (i32.const 0))
(global $sample (export "sample") (mut i32) (i32.const 0))
(global $voice (mut i32) (i32.const 0))
(global $voicesRemain (mut i32) (i32.const 0))
(global $randseed (mut i32) (i32.const 1))
(global $sp (mut i32) (i32.const {{index .Labels "su_stack"}}))
(global $outputBufPtr (export "outputBufPtr") (mut i32) (i32.const {{index .Labels "su_outputbuffer"}}))
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

{{if .HasOp "sync"}}
;;------------------------------------------------------------------------------
;; Sync
;;------------------------------------------------------------------------------

{{- .Align}}
{{- .SetBlockLabel "su_syncbuf"}}
{{- .Block (int (mul 8 .Song.Patch.NumSyncs)) }}

(global $syncbufstart (export "sync") i32 (i32.const {{index .Labels "su_syncbuf"}}))
(global $sync (mut i32) (i32.const {{index .Labels "su_syncbuf"}}))

(func $su_op_sync (param $stereo i32)
    {{- if .Stereo "sync"}}
        (if (local.get $stereo) (then
            (f32.store (global.get $sync) (call $peek2))
            (global.set $sync (i32.add (i32.const 4) (global.get $sync)))
        ))
    {{- end}}
    (f32.store (global.get $sync) (call $peek))
    (global.set $sync (i32.add (i32.const 4) (global.get $sync)))
)
{{end}}

;;------------------------------------------------------------------------------
;; "Entry point" for the player
;;------------------------------------------------------------------------------

(func $render_128_samples (param) (result i32)
(local $rendersamplecount i32) 
(local $should_update_voices i32)
{{- if  .Output16Bit }} (local $channel i32) {{- end }}
    (i32.const 0)
    (local.set $rendersamplecount)
    (i32.const 0)
    (local.set $should_update_voices)    
    (loop $sample_loop
        (if (i32.eq (global.get $sample) (i32.const 0))
            (then
                (i32.const 1)
                (local.set $should_update_voices)
            )
        )
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
{{- if .HasOp "sync"}}
        (global.set $sync (i32.const {{index .Labels "su_syncbuf"}}))
{{- end}}
        (call $su_run_vm)
        {{- template "output_sound.wat" .}}
        (global.set $sample (i32.add (global.get $sample) (i32.const 1)))
        (global.set $globaltick (i32.add (global.get $globaltick) (i32.const 1)))
        (if (i32.eq (global.get $sample) (i32.const {{.Song.SamplesPerRow}}))
            (then
                (global.set $sample (i32.const 0))
                (global.set $row (i32.add (global.get $row) (i32.const 1)))
            )
        )        
        (if (i32.eq (global.get $row) (i32.const {{.PatternLength}}))
            ( then
                (global.set $row (i32.const 0))
                (global.set $pattern (i32.add (global.get $pattern) (i32.const 1)))
            )
        )
        (if (i32.eq (global.get $pattern) (i32.const {{.SequenceLength}}))
            ( then
                (global.set $pattern (i32.const 0))
                (global.set $globaltick (i32.const 0))
                (global.set $outputBufPtr (i32.const {{index .Labels "su_outputbuffer"}}))
            )
        )
        (local.set $rendersamplecount (i32.add (local.get $rendersamplecount) (i32.const 1)))
        (br_if $sample_loop (i32.lt_s (local.get $rendersamplecount) (i32.const 128)))
    )
    (local.get $should_update_voices)
)
(export "render_128_samples" (func $render_128_samples))
(export "update_voices" (func $su_update_voices))

{{- if  .RenderOnStart }}
(start $render) ;; we run render automagically when the module is instantiated
{{- else}}
(export "render" (func $render))
{{- end}}

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
{{- if .HasOp "sync"}}
                (global.set $sync (i32.const {{index .Labels "su_syncbuf"}}))
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
                (i32.const 1)
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
                (i32.store8 offset={{index .Labels "su_trackcurrentvoice"}} (local.get $tracksRemaining) (local.get $voiceNo))
            ))
        ))
        (local.set $si (i32.add (local.get $si) (i32.const {{.SequenceLength}})))
        (br_if $track_loop (local.tee $tracksRemaining (i32.sub (local.get $tracksRemaining) (i32.const 1))))
    end
)

{{- else}}
;; the simple implementation of update_voices: each track has exactly one voice
(func $su_update_voices 
    (local $si i32)
    (local $di i32)
    (local $tracksRemaining i32)
    (local $note i32)
    (local $channel i32)
    (local.set $channel (i32.const 0))
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
            (if (i32.eq (i32.const 0) 
                (i32.load8_u offset={{index .Labels "su_hold_override"}} (local.get $channel))
            )
            (then
                (i32.store offset=4 (local.get $di) (i32.const 1)) ;; release the note
                (if (i32.gt_u (local.get $note) (i32.const {{.Hold}}))(then
                    (memory.fill (local.get $di) (i32.const 0) (i32.const 4096))
                    (i32.store (local.get $di) (local.get $note))
                ))
            ))
        ))
        (local.set $di (i32.add (local.get $di) (i32.const 4096)))
        (local.set $si (i32.add (local.get $si) (i32.const {{.SequenceLength}})))
        (local.set $channel (i32.add (local.get $channel) (i32.const 1)))
        (br_if $track_loop (local.tee $tracksRemaining (i32.sub (local.get $tracksRemaining) (i32.const 1))))
    end
)
{{- end}}

(func $update_single_voice (param $voice_no i32) (param $value i32)
    (local $di i32)
    (local.set $di (
        i32.add (i32.const {{index .Labels "su_voices"}})
        (i32.mul (i32.const 4096) (local.get $voice_no))
    ))
    (i32.store offset=4 (local.get $di) (i32.const 1)) ;; release the note
    (if (i32.gt_u (local.get $value) (i32.const {{.Hold}}))(then
        (memory.fill (local.get $di) (i32.const 0) (i32.const 4096))
        (i32.store (local.get $di) (local.get $value))
    ))
    (i32.store (i32.add (i32.const {{index .Labels "su_hold_override"}}) (local.get $voice_no)) (local.get $value))
)
(export "update_single_voice" (func $update_single_voice))
{{template "patch.wat" .}}

;; All data is collected into a byte buffer and emitted at once
(data (i32.const 0) "{{range .Data}}\{{. | printf "%02x"}}{{end}}")
(data (i32.const {{index .Labels "math_pow_1"}}) "\be\f3\f8y\eca\f6?\190\96[\c6\fe\de\bf=\88\afJ\edq\f5?\a4\fc\d42h\0b\db\bf\b0\10\f0\f09\95\f4?{\b7\1f\n\8bA\d7\bf\85\03\b8\b0\95\c9\f3?{\cfm\1a\e9\9d\d3\bf\a5d\88\0c\19\0d\f3?1\b6\f2\f3\9b\1d\d0\bf\a0\8e\0b{\"^\f2?\f0z;\1b\1d|\c9\bf?4\1aJJ\bb\f1?\9f<\af\93\e3\f9\c2\bf\ba\e5\8a\f0X#\f1?\\\8dx\bf\cb`\b9\bf\a7\00\99A?\95\f0?\ce_G\b6\9do\aa\bf\00\00\00\00\00\00\f0?\00\00\00\00\00\00\00\00\acG\9a\fd\8c`\ee?=\f5$\9f\ca8\b3?\a0j\02\1f\b3\a4\ec?\ba\918T\a9v\c4?\e6\fcjW6 \eb?\d2\e4\c4J\0b\84\ce?-\aa\a1c\d1\c2\e9?\1ce\c6\f0E\06\d4?\edAx\03\e6\86\e8?\f8\9f\1b,\9c\8e\d8?bHS\f5\dcg\e7?\cc{\b1N\a4\e0\dc?")
(data (i32.const {{index .Labels "math_pow_2"}}) "\f0?t\85\15\d3\b0\d9\ef?\0f\89\f9lX\b5\ef?Q[\12\d0\01\93\ef?{Q}<\b8r\ef?\aa\b9h1\87T\ef?8bunz8\ef?\e1\de\1f\f5\9d\1e\ef?\15\b71\n\fe\06\ef?\cb\a9:7\a7\f1\ee?\"4\12L\a6\de\ee?-\89a`\08\ce\ee?\'*6\d5\da\bf\ee?\82O\9dV+\b4\ee?)TH\dd\07\ab\ee?\85U:\b0~\a4\ee?\cd;\7ff\9e\a0\ee?t_\ec\e8u\9f\ee?\87\01\ebs\14\a1\ee?\13\ceL\99\89\a5\ee?\db\a0*B\e5\ac\ee?\e5\c5\cd\b07\b7\ee?\90\f0\a3\82\91\c4\ee?]%>\b2\03\d5\ee?\ad\d3Z\99\9f\e8\ee?G^\fb\f2v\ff\ee?\9cR\85\dd\9b\19\ef?i\90\ef\dc 7\ef?\87\a4\fb\dc\18X\ef?_\9b{3\97|\ef?\da\90\a4\a2\af\a4\ef?@En[v\d0\ef?")

;;(data (i32.const 8388610) "\52\49\46\46\b2\eb\0c\20\57\41\56\45\66\6d\74\20\12\20\20\20\03\20\02\20\44\ac\20\20\20\62\05\20\08\20\20\20\20\20\66\61\63\74\04\20\20\20\e0\3a\03\20\64\61\74\61\80\eb\0c\20")

) ;; END MODULE
