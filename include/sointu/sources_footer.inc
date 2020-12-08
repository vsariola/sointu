;-------------------------------------------------------------------------------
;   ENVELOPE opcode: pushes an ADSR envelope value on stack [0,1]
;-------------------------------------------------------------------------------
;   Mono:   push the envelope value on stack
;   Stereo: push the envelope valeu on stack twice
;-------------------------------------------------------------------------------
%if ENVELOPE_ID > -1

SECT_TEXT(suenvelo)

EXPORT MANGLE_FUNC(su_op_envelope,0)
%ifdef INCLUDE_STEREO_ENVELOPE
    jnc     su_op_envelope_mono
    call    su_op_envelope_mono
    fld     st0
    ret
su_op_envelope_mono:
%endif
    mov     eax, dword [INP-su_voice.inputs+su_voice.release] ; eax = su_instrument.release
    test    eax, eax                            ; if (eax == 0)
    je      su_op_envelope_process              ;   goto process
    mov     dword [WRK+su_env_work.state], ENV_STATE_RELEASE  ; [state]=RELEASE
su_op_envelope_process:
    mov     eax, dword [WRK+su_env_work.state]  ; al=[state]
    fld     dword [WRK+su_env_work.level]       ; x=[level]
    cmp     al, ENV_STATE_SUSTAIN               ; if (al==SUSTAIN)
    je      short su_op_envelope_leave2         ;   goto leave2
su_op_envelope_attac:
    cmp     al, ENV_STATE_ATTAC                 ; if (al!=ATTAC)
    jne     short su_op_envelope_decay          ;   goto decay
    call    su_nonlinear_map                    ; a x, where a=attack
    faddp   st1, st0                            ; a+x
    fld1                                        ; 1 a+x
    fucomi  st1                                 ; if (a+x<=1) // is attack complete?
    fcmovnb st0, st1                            ;   a+x a+x
    jbe     short su_op_envelope_statechange    ; else goto statechange
su_op_envelope_decay:
    cmp     al, ENV_STATE_DECAY                 ; if (al!=DECAY)
    jne     short su_op_envelope_release        ;   goto release
    call    su_nonlinear_map                    ; d x, where d=decay
    fsubp   st1, st0                            ; x-d
    fld     dword [INP+su_env_ports.sustain]    ; s x-d, where s=sustain
    fucomi  st1                                 ; if (x-d>s) // is decay complete?
    fcmovb  st0, st1                            ;   x-d x-d
    jnc     short su_op_envelope_statechange    ; else goto statechange
su_op_envelope_release:
    cmp     al, ENV_STATE_RELEASE               ; if (al!=RELEASE)
    jne     short su_op_envelope_leave          ;   goto leave
    call    su_nonlinear_map                    ; r x, where r=release
    fsubp   st1, st0                            ; x-r
    fldz                                        ; 0 x-r
    fucomi  st1                                 ; if (x-r>0) // is release complete?
    fcmovb  st0, st1                            ;   x-r x-r, then goto leave
    jc      short su_op_envelope_leave
su_op_envelope_statechange:
    inc     dword [WRK+su_env_work.state]       ; [state]++
su_op_envelope_leave:
    fstp    st1                                 ; x', where x' is the new value
    fst     dword [WRK+su_env_work.level]       ; [level]=x'
su_op_envelope_leave2:
    fmul    dword [INP+su_env_ports.gain]       ; [gain]*x'
    ret

%endif ; SU_USE_ENVELOPE

;-------------------------------------------------------------------------------
;   NOISE opcode: creates noise
;-------------------------------------------------------------------------------
;   Mono:   push a random value [-1,1] value on stack
;   Stereo: push two (differeent) random values on stack
;-------------------------------------------------------------------------------
%if NOISE_ID > -1

SECT_TEXT(sunoise)

EXPORT MANGLE_FUNC(su_op_noise,0)
    mov     _CX,_SP
%ifdef INCLUDE_STEREO_NOISE
    jnc     su_op_noise_mono
    call    su_op_noise_mono
su_op_noise_mono:
%endif
    imul    eax, [_CX + su_stack.randseed],16007
    mov     [_CX + su_stack.randseed],eax
    fild    dword [_CX + su_stack.randseed]
 do fidiv   dword [,c_RandDiv,]
    fld     dword [INP+su_noise_ports.shape]
    call    su_waveshaper
    fld     dword [INP+su_noise_ports.gain]
    fmulp   st1, st0
    ret

%define SU_INCLUDE_WAVESHAPER

%endif

;-------------------------------------------------------------------------------
;   OSCILLAT opcode: oscillator, the heart of the synth
;-------------------------------------------------------------------------------
;   Mono:   push oscillator value on stack
;   Stereo: push l r on stack, where l has opposite detune compared to r
;-------------------------------------------------------------------------------
%if OSCILLAT_ID > -1

SECT_TEXT(suoscill)

EXPORT MANGLE_FUNC(su_op_oscillat,0)
    lodsb                                   ; load the flags
%ifdef RUNTIME_TABLES  
    %ifdef INCLUDE_SAMPLES
    mov     _DI, [_SP + su_stack.sampleoffs]; we need to put this in a register, as the stereo & unisons screw the stack positions
    %endif                                  ; ain't we lucky that _DI was unused throughout
%endif
    fld     dword [INP+su_osc_ports.detune] ; e, where e is the detune [0,1]
 do fsub    dword [,c_0_5,]                 ; e-.5
    fadd    st0, st0                        ; d=2*e-.5, where d is the detune [-1,1]
%ifdef INCLUDE_STEREO_OSCILLAT
    jnc     su_op_oscillat_mono
    fld     st0                             ; d d
    call    su_op_oscillat_mono             ; r d
    add     WRK, 4                          ; state vars: r1 l1 r2 l2 r3 l3 r4 l4, for the unison osc phases
    fxch                                    ; d r
    fchs                                    ; -d r, negate the detune for second round
    su_op_oscillat_mono:
%endif
%ifdef INCLUDE_UNISONS
    push_registers _AX, WRK, _AX
    fldz                            ; 0 d
    fxch                            ; d a=0, "accumulated signal"
su_op_oscillat_unison_loop:
    fst     dword [_SP]             ; save the current detune, d. We could keep it in fpu stack but it was getting big.
    call    su_op_oscillat_single   ; s a
    faddp   st1, st0                ; a+=s
    test    al, UNISONBITS
    je      su_op_oscillat_unison_out
    add     WRK, 8
    fld     dword [INP+su_osc_ports.phaseofs] ; p s
 do fadd    dword [,c_i12,]                   ; p s, add some little phase offset to unison oscillators so they don't start in sync
    fstp    dword [INP+su_osc_ports.phaseofs] ; s    note that this changes the phase for second, possible stereo run. That's probably ok
    fld     dword [_SP]             ; d s
 do fmul    dword [,c_0_5,]         ; .5*d s    // negate and halve the detune of each oscillator
    fchs                            ; -.5*d s   // negate and halve the detune of each oscillator
    dec     eax
    jmp     short su_op_oscillat_unison_loop
su_op_oscillat_unison_out:
    pop_registers _AX, WRK, _AX
    ret
su_op_oscillat_single:
%endif
    fld     dword [INP+su_osc_ports.transpose]
 do fsub    dword [,c_0_5,]
 do fdiv    dword [,c_i128,]
    faddp   st1
    test    al, byte LFOFLAG
    jnz     su_op_oscillat_skipnote
    fiadd   dword [INP-su_voice.inputs+su_voice.note]   ; // st0 is note, st1 is t+d offset
su_op_oscillat_skipnote:
 do fmul    dword [,c_i12,]
    call    MANGLE_FUNC(su_power,0)
    test    al, byte LFOFLAG
    jz      short su_op_oscillat_normalize_note
 do fmul    dword [,c_lfo_normalize,]  ; // st0 is now frequency for lfo
    jmp     short su_op_oscillat_normalized
su_op_oscillat_normalize_note:
 do fmul    dword [,c_freq_normalize,]   ; // st0 is now frequency
su_op_oscillat_normalized:
    fadd    dword [WRK+su_osc_wrk.phase]
    fst     dword [WRK+su_osc_wrk.phase]
    fadd    dword [INP+su_osc_ports.phaseofs]
%ifdef INCLUDE_SAMPLES
    test    al, byte SAMPLE
    jz      short su_op_oscillat_not_sample
    call    su_oscillat_sample
    jmp     su_op_oscillat_shaping ; skip the rest to avoid color phase normalization and colorloading
su_op_oscillat_not_sample:
%endif
    fld1
    fadd    st1, st0
    fxch
    fprem
    fstp    st1
    fld     dword [INP+su_osc_ports.color]               ; // c      p
    ; every oscillator test included if needed
%ifdef INCLUDE_SINE
    test    al, byte SINE
    jz      short su_op_oscillat_notsine
    call    su_oscillat_sine
su_op_oscillat_notsine:
%endif
%ifdef INCLUDE_TRISAW
    test    al, byte TRISAW
    jz      short su_op_oscillat_not_trisaw
    call    su_oscillat_trisaw
su_op_oscillat_not_trisaw:
%endif
%ifdef INCLUDE_PULSE
    test    al, byte PULSE
    jz      short su_op_oscillat_not_pulse
    call    su_oscillat_pulse
su_op_oscillat_not_pulse:
%endif
%ifdef INCLUDE_GATE
    test    al, byte GATE
    jz      short su_op_oscillat_not_gate
    call    su_oscillat_gate
    jmp     su_op_oscillat_gain ; skip waveshaping as the shape parameter is reused for gateshigh
su_op_oscillat_not_gate:
%endif
su_op_oscillat_shaping:
    ; finally, shape the oscillator and apply gain
    fld     dword [INP+su_osc_ports.shape]
    call    su_waveshaper
su_op_oscillat_gain:
    fld     dword [INP+su_osc_ports.gain]
    fmulp   st1, st0
    ret
    %define SU_INCLUDE_WAVESHAPER

%endif

; PULSE
%ifdef INCLUDE_PULSE

SECT_TEXT(supulse)

su_oscillat_pulse:
    fucomi  st1                             ; // c      p
    fld1
    jnc     short su_oscillat_pulse_up     ; // +1     c       p
    fchs                                    ; // -1     c       p
su_oscillat_pulse_up:
    fstp    st1                             ; // +-1    p
    fstp    st1                             ; // +-1
    ret

%endif

; TRISAW
%ifdef INCLUDE_TRISAW

SECT_TEXT(sutrisaw)

su_oscillat_trisaw:
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
%endif

; SINE
%ifdef INCLUDE_SINE

SECT_TEXT(susine)

su_oscillat_sine:
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

%endif

%ifdef INCLUDE_GATE

SECT_TEXT(sugate)

su_oscillat_gate:
    fxch                                    ; p c
    fstp    st1                             ; p
 do fmul    dword [,c_16,]                  ; 16*p
    push    _AX
    push    _AX
    fistp   dword [_SP]                     ; s=int(16*p), stack empty
    fld1                                    ; 1
    pop     _AX
    and     al, 0xf                         ; ax=int(16*p) & 15, stack: 1
    bt      word [VAL-4],ax                 ; if bit ax of the gate word is set
    jc      go4kVCO_gate_bit                ;   goto gate_bit
    fsub    st0, st0                        ; stack: 0
go4kVCO_gate_bit:                           ; stack: 0/1, let's call it x
    fld     dword [WRK+su_osc_wrk.gatestate] ; g x, g is gatestate, x is the input to this filter 0/1
    fsub    st1                             ; g-x x
 do fmul    dword [,c_dc_const,]            ; c(g-x) x
    faddp   st1, st0                        ; x+c(g-x)
    fst     dword [WRK+su_osc_wrk.gatestate]; g'=x+c(g-x)
    pop     _AX                             ; Another way to see this (c~0.996)
    ret                                     ; g'=cg+(1-c)x
    ; This is a low-pass to smooth the gate transitions

%define USE_C_16
%define USE_C_DC_CONST

%endif

; SAMPLES
%ifdef INCLUDE_SAMPLES

SECT_TEXT(suoscsam)

su_oscillat_sample:                                         ; p
    push_registers _AX,_DX,_CX,_BX                              ; edx must be saved, eax & ecx if this is stereo osc
    push    _AX
    mov     al, byte [VAL-4]                                ; reuse "color" as the sample number
%ifdef RUNTIME_TABLES                                       ; when using RUNTIME_TABLES, assumed the sample_offset ptr is in _DI
 do{lea     _DI, [}, _DI, _AX*8,]                           ; edi points now to the sample table entry
%else
 do{lea     _DI, [}, MANGLE_DATA(su_sample_offsets), _AX*8,]; edi points now to the sample table entry
%endif
 do fmul    dword [,c_samplefreq_scaling,]                  ; p*r
    fistp   dword [_SP]
    pop     _DX                                             ; edx is now the sample number
    movzx   ebx, word [_DI + su_sample_offset.loopstart]    ; ecx = loopstart
    sub     edx, ebx                                        ; if sample number < loop start
    jl      su_oscillat_sample_not_looping                  ;   then we're not looping yet
    mov     eax, edx                                        ; eax = sample number
    movzx   ecx, word [_DI + su_sample_offset.looplength]   ; edi is now the loop length
    xor     edx, edx                                        ; div wants edx to be empty
    div     ecx                                             ; edx is now the remainder
su_oscillat_sample_not_looping:
    add     edx, ebx                                        ; sampleno += loopstart
    add     edx, dword [_DI + su_sample_offset.start]
 do fild    word [,MANGLE_DATA(su_sample_table),_DX*2,]
 do fdiv    dword [,c_32767,]
    pop_registers _AX,_DX,_CX,_BX
    ret

%define USE_C_32767
%define USE_C_SAMPLEFREQ_SCALING

%endif

;-------------------------------------------------------------------------------
;   LOADVAL opcode
;-------------------------------------------------------------------------------
;   Mono: push 2*v-1 on stack, where v is the input to port "value"
;   Stereo: push 2*v-1 twice on stack
;-------------------------------------------------------------------------------
%if LOADVAL_ID > -1

SECT_TEXT(suloadvl)

EXPORT MANGLE_FUNC(su_op_loadval,0)
%ifdef INCLUDE_STEREO_LOADVAL
    jnc     su_op_loadval_mono
    call    su_op_loadval_mono
su_op_loadval_mono:
%endif
    fld     dword [INP+su_load_val_ports.value] ; v
 do fsub    dword [,c_0_5,]
    fadd    st0                                 ; 2*v-1
    ret

%endif ; SU_USE_LOAD_VAL


;-------------------------------------------------------------------------------
;   RECEIVE opcode
;-------------------------------------------------------------------------------
;   Mono:   push l on stack, where l is the left channel received
;   Stereo: push l r on stack
;-------------------------------------------------------------------------------
%if RECEIVE_ID > -1

SECT_TEXT(sureceiv)

EXPORT MANGLE_FUNC(su_op_receive,0)
    lea     _DI, [WRK+su_unit.ports]
%ifdef INCLUDE_STEREO_RECEIVE
    jnc     su_op_receive_mono
    xor     ecx,ecx
    fld     dword [_DI+su_receive_ports.right]
    mov     dword [_DI+su_receive_ports.right],ecx
su_op_receive_mono:
%endif
    xor     ecx,ecx
    fld     dword [_DI+su_receive_ports.left]
    mov     dword [_DI+su_receive_ports.left],ecx
    ret

%endif ; RECEIVE_ID > -1

;-------------------------------------------------------------------------------
;   IN opcode: inputs and clears a global port
;-------------------------------------------------------------------------------
;   Mono: push the left channel of a global port (out or aux)
;   Stereo: also push the right channel (stack in l r order)
;-------------------------------------------------------------------------------
%if IN_ID > -1

SECT_TEXT(suopin)

EXPORT MANGLE_FUNC(su_op_in,0)
    lodsb
    mov     _DI, [_SP + su_stack.synth]
    %ifdef INCLUDE_STEREO_IN               
        jnc     su_op_in_mono                
        xor     ecx, ecx ; we cannot xor before jnc, so we have to do it mono & stereo. LAHF / SAHF could do it, but is the same number of bytes with more entropy
        fld     dword [_DI + su_synthworkspace.right + _AX*4]
        mov     dword [_DI + su_synthworkspace.right + _AX*4], ecx        
    su_op_in_mono:        
    %endif
    xor     ecx, ecx        
    fld     dword [_DI + su_synthworkspace.left + _AX*4]
    mov     dword [_DI + su_synthworkspace.left + _AX*4], ecx            
    ret

%endif ; SU_IN_ID > -1
