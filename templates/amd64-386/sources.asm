{{if .HasOp "envelope" -}}
;-------------------------------------------------------------------------------
;   ENVELOPE opcode: pushes an ADSR envelope value on stack [0,1]
;-------------------------------------------------------------------------------
;   Mono:   push the envelope value on stack
;   Stereo: push the envelope valeu on stack twice
;-------------------------------------------------------------------------------
{{.Func "su_op_envelope" "Opcode"}}
{{- if .StereoAndMono "envelope"}}
    jnc     su_op_envelope_mono
{{- end}}
{{- if .Stereo "envelope"}}
    call    su_op_envelope_mono
    fld     st0
    ret
su_op_envelope_mono:
{{- end}}
    mov     eax, dword [{{.INP}}-su_voice.inputs+su_voice.release] ; eax = su_instrument.release
    test    eax, eax                            ; if (eax == 0)
    je      su_op_envelope_process              ;   goto process
    mov     al, {{.InputNumber "envelope" "release"}}  ; [state]=RELEASE
    mov     dword [{{.WRK}}], eax               ; note that mov al, XXX; mov ..., eax is less bytes than doing it directly
su_op_envelope_process:
    mov     eax, dword [{{.WRK}}]  ; al=[state]
    fld     dword [{{.WRK}}+4]       ; x=[level]
    cmp     al, {{.InputNumber "envelope" "sustain"}}               ; if (al==SUSTAIN)
    je      short su_op_envelope_leave2         ;   goto leave2
su_op_envelope_attac:
    cmp     al, {{.InputNumber "envelope" "attack"}}                 ; if (al!=ATTAC)
    jne     short su_op_envelope_decay          ;   goto decay
    {{.Call "su_nonlinear_map"}}                ; a x, where a=attack
    faddp   st1, st0                            ; a+x
    fld1                                        ; 1 a+x
    fucomi  st1                                 ; if (a+x<=1) // is attack complete?
    fcmovnb st0, st1                            ;   a+x a+x
    jbe     short su_op_envelope_statechange    ; else goto statechange
su_op_envelope_decay:
    cmp     al, {{.InputNumber "envelope" "decay"}}                 ; if (al!=DECAY)
    jne     short su_op_envelope_release        ;   goto release
    {{.Call "su_nonlinear_map"}}                ; d x, where d=decay
    fsubp   st1, st0                            ; x-d
    fld     dword [{{.Input "envelope" "sustain"}}]    ; s x-d, where s=sustain
    fucomi  st1                                 ; if (x-d>s) // is decay complete?
    fcmovb  st0, st1                            ;   x-d x-d
    jnc     short su_op_envelope_statechange    ; else goto statechange
su_op_envelope_release:
    cmp     al, {{.InputNumber "envelope" "release"}}               ; if (al!=RELEASE)
    jne     short su_op_envelope_leave          ;   goto leave
    {{.Call "su_nonlinear_map"}}                ; r x, where r=release
    fsubp   st1, st0                            ; x-r
    fldz                                        ; 0 x-r
    fucomi  st1                                 ; if (x-r>0) // is release complete?
    fcmovb  st0, st1                            ;   x-r x-r, then goto leave
    jc      short su_op_envelope_leave
su_op_envelope_statechange:
    inc     dword [{{.WRK}}]       ; [state]++
su_op_envelope_leave:
    fstp    st1                                 ; x', where x' is the new value
    fst     dword [{{.WRK}}+4]       ; [level]=x'
su_op_envelope_leave2:
    fmul    dword [{{.Input "envelope" "gain"}}]       ; [gain]*x'
    ret
{{end}}


{{- if .HasOp "noise"}}
;-------------------------------------------------------------------------------
;   NOISE opcode: creates noise
;-------------------------------------------------------------------------------
;   Mono:   push a random value [-1,1] value on stack
;   Stereo: push two (differeent) random values on stack
;-------------------------------------------------------------------------------
{{.Func "su_op_noise" "Opcode"}}
    lea     {{.CX}},[{{.Stack "RandSeed"}}]
{{- if .StereoAndMono "noise"}}
    jnc     su_op_noise_mono
{{- end}}
{{- if .Stereo "noise"}}
    call    su_op_noise_mono
su_op_noise_mono:
{{- end}}
    imul    eax, [{{.CX}}],16007
    mov     [{{.CX}}],eax
    fild    dword [{{.CX}}]
{{- .Prepare (.Int 2147483648)}}
    fidiv   dword [{{.Use (.Int 2147483648)}}] ; 65536*32768
    fld     dword [{{.Input "noise" "shape"}}]
    {{.Call "su_waveshaper"}}
    fld     dword [{{.Input "noise" "gain"}}]
    fmulp   st1, st0
    ret
{{end}}


{{- if .HasOp "oscillator"}}
;-------------------------------------------------------------------------------
;   OSCILLAT opcode: oscillator, the heart of the synth
;-------------------------------------------------------------------------------
;   Mono:   push oscillator value on stack
;   Stereo: push l r on stack, where l has opposite detune compared to r
;-------------------------------------------------------------------------------
{{.Func "su_op_oscillator" "Opcode"}}
    lodsb                                   ; load the flags
{{- if .Library}}
    mov     {{.DI}}, [{{.Stack "SampleTable"}}]; we need to put this in a register, as the stereo & unisons screw the stack positions
                                 ; ain't we lucky that {{.DI}} was unused throughout
{{- end}}
    fld     dword [{{.Input "oscillator" "detune"}}] ; e, where e is the detune [0,1]
{{- .Prepare (.Float 0.5)}}
    fsub    dword [{{.Use (.Float 0.5)}}]                 ; e-.5
    fadd    st0, st0                        ; d=2*e-.5, where d is the detune [-1,1]
{{- if .StereoAndMono "oscillator"}}
    jnc     su_op_oscillat_mono
{{- end}}
{{- if .Stereo "oscillator"}}
    fld     st0                             ; d d
    call    su_op_oscillat_mono             ; r d
    ;; WARNING: this is a bug. WRK should be nonvolatile, but we are changing it. It does not cause immediate problems but modulations will be off.
    ;; Figure out how to do this; maybe $WRK should be volatile (pushed by the virtual machine)
    add     {{.WRK}}, 4                     ; state vars: r1 l1 r2 l2 r3 l3 r4 l4, for the unison osc phases-
    fxch                                    ; d r
    fchs                                    ; -d r, negate the detune for second round
su_op_oscillat_mono:
{{- end}}
{{- if .SupportsParamValueOtherThan "oscillator" "unison" 0}}
    {{.PushRegs .AX "" .WRK "OscWRK" .AX "OscFlags"}}
    fldz                            ; 0 d
    fxch                            ; d a=0, "accumulated signal"
su_op_oscillat_unison_loop:
    fst     dword [{{.SP}}]             ; save the current detune, d. We could keep it in fpu stack but it was getting big.
    call    su_op_oscillat_single   ; s a
    faddp   st1, st0                ; a+=s
    test    al, 3
    je      su_op_oscillat_unison_out
    ;; WARNING: this is a bug. WRK should be nonvolatile, but we are changing it. It does not cause immediate problems but modulations will be off.
    ;; Figure out how to do this; maybe $WRK should be volatile (pushed by the virtual machine)
    add     {{.WRK}}, 8
    fld     dword [{{.Input "oscillator" "phase"}}] ; p s
{{.Int 0x3DAAAAAA | .Prepare}}
    fadd    dword [{{.Int 0x3DAAAAAA | .Use}}]  ; 1/12 p s, add some little phase offset to unison oscillators so they don't start in sync
    fstp    dword [{{.Input "oscillator" "phase"}}] ; s    note that this changes the phase for second, possible stereo run. That's probably ok
    fld     dword [{{.SP}}]             ; d s
{{.Float 0.5 | .Prepare}}
    fmul    dword [{{.Float 0.5 | .Use}}]         ; .5*d s    // negate and halve the detune of each oscillator
    fchs                            ; -.5*d s   // negate and halve the detune of each oscillator
    dec     eax
    jmp     short su_op_oscillat_unison_loop
su_op_oscillat_unison_out:
    {{.PopRegs .AX .WRK .AX}}
    ret
su_op_oscillat_single:
{{- end}}
    fld     dword [{{.Input "oscillator" "transpose"}}]
{{- .Float 0.5 | .Prepare}}
    fsub    dword [{{.Float 0.5 | .Use}}]
{{- .Float 0.0078125 | .Prepare}}
    fdiv    dword [{{.Float 0.0078125 | .Use}}]
    faddp   st1
    test    al, byte 0x08
    jnz     su_op_oscillat_skipnote
    fiadd   dword [{{.INP}}-su_voice.inputs+su_voice.note]   ; // st0 is note, st1 is t+d offset
su_op_oscillat_skipnote:
{{- .Int 0x3DAAAAAA | .Prepare}}
    fmul    dword [{{.Int 0x3DAAAAAA | .Use}}]
    {{.Call "su_power"}}
    test    al, byte 0x08
    jz      short su_op_oscillat_normalize_note
{{- .Float 0.000038 | .Prepare}}
    fmul    dword [{{.Float 0.000038 | .Use}}]  ; // st0 is now frequency for lfo
    jmp     short su_op_oscillat_normalized
su_op_oscillat_normalize_note:
{{- .Float 0.000092696138 | .Prepare}}
    fmul    dword [{{.Float 0.000092696138 | .Use}}]   ; // st0 is now frequency
su_op_oscillat_normalized:
    fadd    dword [{{.WRK}}]
{{- if .SupportsParamValue "oscillator" "type" .Sample}}
    test    al, byte 0x80
    jz      short su_op_oscillat_not_sample
    fst     dword [{{.WRK}}]  ; for samples, we store the phase without mod(p,1)
{{- if or (.SupportsParamValueOtherThan "oscillator" "phase" 0) (.SupportsModulation "oscillator" "phase")}}
    fadd    dword [{{.Input "oscillator" "phase"}}]
{{- end}}
    {{.Call "su_oscillat_sample"}}
    jmp     su_op_oscillat_shaping ; skip the rest to avoid color phase normalization and colorloading
su_op_oscillat_not_sample:
{{- end}}
    fld1                     ; we need to take mod(p,1) so the frequency does not drift as the float
    fadd    st1, st0         ; make no mistake: without this, there is audible drifts in oscillator pitch
    fxch                     ; as the actual period changes once the phase becomes too big
    fprem                    ; we actually computed mod(p+1,1) instead of mod(p,1) as the fprem takes mod
    fstp    st1              ; towards zero
    fst     dword [{{.WRK}}] ; store back the updated phase
{{- if or (.SupportsParamValueOtherThan "oscillator" "phase" 0) (.SupportsModulation "oscillator" "phase")}}
    fadd    dword [{{.Input "oscillator" "phase"}}]
    fld1                    ; this is a bit stupid, but we need to take mod(x,1) again after phase modulations
    fadd    st1, st0        ; as the actual oscillator functions expect x in [0,1]
    fxch
    fprem
    fstp    st1
{{- end}}
    fld     dword [{{.Input "oscillator" "color"}}]               ; // c      p
    ; every oscillator test included if needed
{{- if .SupportsParamValue "oscillator" "type" .Sine}}
    test    al, byte 0x40
    jz      short su_op_oscillat_notsine
    {{.Call "su_oscillat_sine"}}
su_op_oscillat_notsine:
{{- end}}
{{- if .SupportsParamValue "oscillator" "type" .Trisaw}}
    test    al, byte 0x20
    jz      short su_op_oscillat_not_trisaw
    {{.Call "su_oscillat_trisaw"}}
su_op_oscillat_not_trisaw:
{{- end}}
{{- if .SupportsParamValue "oscillator" "type" .Pulse}}
    test    al, byte 0x10
    jz      short su_op_oscillat_not_pulse
    {{.Call "su_oscillat_pulse"}}
su_op_oscillat_not_pulse:
{{- end}}
{{- if .SupportsParamValue "oscillator" "type" .Gate}}
    test    al, byte 0x04
    jz      short su_op_oscillat_not_gate
    {{.Call "su_oscillat_gate"}}
    jmp     su_op_oscillat_gain ; skip waveshaping as the shape parameter is reused for gateshigh
su_op_oscillat_not_gate:
{{- end}}
su_op_oscillat_shaping:
    ; finally, shape the oscillator and apply gain
    fld     dword [{{.Input "oscillator" "shape"}}]
    {{.Call "su_waveshaper"}}
su_op_oscillat_gain:
    fld     dword [{{.Input "oscillator" "gain"}}]
    fmulp   st1, st0
    ret
{{end}}


{{- if .HasCall "su_oscillat_pulse"}}
{{.Func "su_oscillat_pulse"}}
    fucomi  st1                             ; // c      p
    fld1
    jnc     short su_oscillat_pulse_up      ; // +1     c       p
    fchs                                    ; // -1     c       p
su_oscillat_pulse_up:
    fstp    st1                             ; // +-1    p
    fstp    st1                             ; // +-1
    ret
{{end}}


{{- if .HasCall "su_oscillat_trisaw"}}
{{.Func "su_oscillat_trisaw"}}
    fucomi  st1                             ; // c      p
    jnc     short su_oscillat_trisaw_up
    fld1                                    ; // 1      c       p
    fsubr   st2, st0                        ; // 1      c       1-p
    fsubrp  st1, st0                        ; // 1-c    1-p
su_oscillat_trisaw_up:
    fdivp   st1, st0                        ; // tp'/tc
    fadd    st0                             ; // 2*''
    fld1                                    ; // 1      2*''
    fsubp   st1, st0                        ; // 2*''-1
    ret
{{end}}


{{- if .HasCall "su_oscillat_sine"}}
{{.Func "su_oscillat_sine"}}
    fucomi  st1                             ; // c      p
    jnc     short su_oscillat_sine_do
    fstp    st1
    fsub    st0, st0                        ; // 0
    ret
su_oscillat_sine_do:
    fdivp   st1, st0                        ; // p/c
    fldpi                                   ; // pi     p
    fadd    st0                             ; // 2*pi   p
    fmulp   st1, st0                        ; // 2*pi*p
    fsin                                    ; // sin(2*pi*p)
    ret
{{end}}


{{- if .HasCall "su_oscillat_gate"}}
{{.Func "su_oscillat_gate"}}
    fxch                                    ; p c
    fstp    st1                             ; p
{{- .Float 16.0 | .Prepare | indent 4}}
    fmul    dword [{{.Float 16.0 | .Use}}]                  ; 16*p
    push    {{.AX}}
    push    {{.AX}}
    fistp   dword [{{.SP}}]                     ; s=int(16*p), stack empty
    fld1                                    ; 1
    pop     {{.AX}}
    and     al, 0xf                         ; ax=int(16*p) & 15, stack: 1
    bt      word [{{.VAL}}-4],ax                 ; if bit ax of the gate word is set
    jc      su_oscillat_gate_bit                ;   goto gate_bit
    fsub    st0, st0                        ; stack: 0
su_oscillat_gate_bit:                           ; stack: 0/1, let's call it x
    fld     dword [{{.WRK}}+16] ; g x, g is gatestate, x is the input to this filter 0/1
    fsub    st1                             ; g-x x
{{- .Float 0.99609375 | .Prepare | indent 4}}
    fmul    dword [{{.Float 0.99609375 | .Use}}]            ; c(g-x) x
    faddp   st1, st0                        ; x+c(g-x)
    fst     dword [{{.WRK}}+16]; g'=x+c(g-x) NOTE THAT UNISON 2 & UNISON 3 ALSO USE {{.WRK}}+16, so gate and unison 2 & 3 don't work. Probably should delete that low pass altogether
    pop     {{.AX}}                             ; Another way to see this (c~0.996)
    ret                                     ; g'=cg+(1-c)x
    ; This is a low-pass to smooth the gate transitions
{{end}}


{{- if .HasCall "su_oscillat_sample"}}
{{.Func "su_oscillat_sample"}}
    {{- .PushRegs .AX "SampleAx" .DX "SampleDx" .CX "SampleCx" .BX "SampleBx" | indent 4}}                              ; edx must be saved, eax & ecx if this is stereo osc
    push    {{.AX}}
    mov     al, byte [{{.VAL}}-4]                                ; reuse "color" as the sample number
{{- if .Library}}
    lea     {{.DI}}, [{{.DI}} + {{.AX}}*8]                           ; edi points now to the sample table entry
{{- else}}
{{- .Prepare "su_sample_offsets" | indent 4}}
    lea     {{.DI}}, [{{.Use "su_sample_offsets"}} + {{.AX}}*8]; edi points now to the sample table entry
{{- end}}
{{- .Float 84.28074964676522 | .Prepare | indent 4}}
    fmul    dword [{{.Float 84.28074964676522 | .Use}}]                  ; p*r
    fistp   dword [{{.SP}}]
    pop     {{.DX}}                                             ; edx is now the sample number
    movzx   ebx, word [{{.DI}} + 4]    ; ecx = loopstart
    sub     edx, ebx                                        ; if sample number < loop start
    jl      su_oscillat_sample_not_looping                  ;   then we're not looping yet
    mov     eax, edx                                        ; eax = sample number
    movzx   ecx, word [{{.DI}} + 6]   ; edi is now the loop length
    xor     edx, edx                                        ; div wants edx to be empty
    div     ecx                                             ; edx is now the remainder
su_oscillat_sample_not_looping:
    add     edx, ebx                                        ; sampleno += loopstart
    add     edx, dword [{{.DI}}]
{{- .Prepare "su_sample_table" | indent 4}}
    fild    word [{{.Use "su_sample_table"}} + {{.DX}}*2]
{{- .Float 32767.0 | .Prepare | indent 4}}
    fdiv    dword [{{.Float 32767.0 | .Use}}]
    {{- .PopRegs .AX .DX .CX .BX | indent 4}}
    ret
{{end}}


{{- if .HasOp "loadval"}}
;-------------------------------------------------------------------------------
;   LOADVAL opcode
;-------------------------------------------------------------------------------
{{- if .Mono "loadval"}}
;   Mono: push 2*v-1 on stack, where v is the input to port "value"
{{- end}}
{{- if .Stereo "loadval"}}
;   Stereo: push 2*v-1 twice on stack
{{- end}}
;-------------------------------------------------------------------------------
{{.Func "su_op_loadval" "Opcode"}}
    {{- if .StereoAndMono "loadval" }}
    jnc     su_op_loadval_mono
    {{- end}}
    {{- if .Stereo "loadval" }}
    call    su_op_loadval_mono
su_op_loadval_mono:
    {{- end }}
    fld     dword [{{.Input "loadval" "value"}}] ; v
{{- .Float 0.5 | .Prepare | indent 4}}
    fsub    dword [{{.Float 0.5 | .Use}}]
    fadd    st0                                 ; 2*v-1
    ret
{{end}}


{{- if .HasOp "receive"}}
;-------------------------------------------------------------------------------
;   RECEIVE opcode
;-------------------------------------------------------------------------------
{{- if .Mono "receive"}}
;   Mono:   push l on stack, where l is the left channel received
{{- end}}
{{- if .Stereo "receive"}}
;   Stereo: push l r on stack
{{- end}}
;-------------------------------------------------------------------------------
{{.Func "su_op_receive" "Opcode"}}
    lea     {{.DI}}, [{{.WRK}}+su_unit.ports]
{{- if .StereoAndMono "receive"}}
    jnc     su_op_receive_mono
{{- end}}
{{- if .Stereo "receive"}}
    xor     ecx,ecx
    fld     dword [{{.DI}}+4]
    mov     dword [{{.DI}}+4],ecx
{{- end}}
{{- if .StereoAndMono "receive"}}
su_op_receive_mono:
    xor     ecx,ecx
{{- end}}
    fld     dword [{{.DI}}]
    mov     dword [{{.DI}}],ecx
    ret
{{end}}


{{- if .HasOp "in"}}
;-------------------------------------------------------------------------------
;   IN opcode: inputs and clears a global port
;-------------------------------------------------------------------------------
;   Mono: push the left channel of a global port (out or aux)
;   Stereo: also push the right channel (stack in l r order)
;-------------------------------------------------------------------------------
{{.Func "su_op_in" "Opcode"}}
    lodsb
    mov     {{.DI}}, [{{.Stack "Synth"}}]
{{- if .StereoAndMono "in"}}
    jnc     su_op_in_mono
{{- end}}
{{- if .Stereo "in"}}
    xor     ecx, ecx ; we cannot xor before jnc, so we have to do it mono & stereo. LAHF / SAHF could do it, but is the same number of bytes with more entropy
    fld     dword [{{.DI}} + su_synthworkspace.right + {{.AX}}*4]
    mov     dword [{{.DI}} + su_synthworkspace.right + {{.AX}}*4], ecx
{{- end}}
{{- if .StereoAndMono "in"}}
su_op_in_mono:
    xor     ecx, ecx
{{- end}}
    fld     dword [{{.DI}} + su_synthworkspace.left + {{.AX}}*4]
    mov     dword [{{.DI}} + su_synthworkspace.left + {{.AX}}*4], ecx
    ret
{{end}}
