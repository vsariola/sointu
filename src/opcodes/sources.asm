;-------------------------------------------------------------------------------
;   ENV Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;   Output:     st0     :   envelope value, [gain]*level. The slopes of
;                           level is 2^(-24*p) per sample, where p is either
;                           attack, decay or release in [0,1] range
;   Dirty:      eax, edx
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
kmENV_func_do:
    mov     eax, dword [ecx+su_unit.size-su_voice.workspace+su_voice.release] ; eax = su_instrument.release
    test    eax, eax                            ; if (eax == 0)
    je      kmENV_func_process                  ;   goto process
    mov     dword [WRK+su_env_work.state], ENV_STATE_RELEASE  ; [state]=RELEASE
kmENV_func_process:
    mov     eax, dword [WRK+su_env_work.state]    ; al=[state]
    fld     dword [WRK+su_env_work.level]         ; x=[level]
    cmp     al, ENV_STATE_SUSTAIN               ; if (al==SUSTAIN)
    je      short kmENV_func_leave2             ;   goto leave2
kmENV_func_attac:
    cmp     al, ENV_STATE_ATTAC                 ; if (al!=ATTAC)
    jne     short kmENV_func_decay              ;   goto decay
    call    su_env_map                          ; a x, where a=attack
    faddp   st1, st0                            ; a+x
    fld1                                        ; 1 a+x
    fucomi  st1                                 ; if (a+x<=1) // is attack complete?
    fcmovnb st0, st1                            ;   a+x a+x
    jbe     short kmENV_func_statechange        ; else goto statechange
kmENV_func_decay:
    cmp     al, ENV_STATE_DECAY                 ; if (al!=DECAY)
    jne     short kmENV_func_release            ;   goto release
    call    su_env_map                          ; d x, where d=decay
    fsubp   st1, st0                            ; x-d
    fld     dword [edx+su_env_ports.sustain]       ; s x-d, where s=sustain
    fucomi  st1                                 ; if (x-d>s) // is decay complete?
    fcmovb  st0, st1                            ;   x-d x-d
    jnc     short kmENV_func_statechange        ; else goto statechange
kmENV_func_release:
    cmp     al, ENV_STATE_RELEASE               ; if (al!=RELEASE)
    jne     short kmENV_func_leave              ;   goto leave
    call    su_env_map                            ; r x, where r=release
    fsubp   st1, st0                            ; x-r
    fldz                                        ; 0 x-r
    fucomi  st1                                 ; if (x-r>0) // is release complete?
    fcmovb  st0, st1                            ;   x-r x-r, then goto leave
    jc      short kmENV_func_leave
kmENV_func_statechange:
    inc     dword [WRK+su_env_work.state]         ; [state]++
kmENV_func_leave:
    fstp    st1                                 ; x', where x' is the new value
    fst     dword [WRK+su_env_work.level]         ; [level]=x'
kmENV_func_leave2:
    fmul    dword [edx+su_env_ports.gain]          ; [gain]*x'
    ret

%endif ; SU_USE_ENVELOPE

;-------------------------------------------------------------------------------
;   su_noise function: noise oscillators
;-------------------------------------------------------------------------------
%if NOISE_ID > -1

SECT_TEXT(sunoise)

EXPORT MANGLE_FUNC(su_op_noise,0)
%ifdef INCLUDE_STEREO_NOISE
    jnc     su_op_noise_mono
    call    su_op_noise_mono
su_op_noise_mono:
%endif
    call    MANGLE_FUNC(FloatRandomNumber,0)
    fld     dword [edx+su_noise_ports.shape]
    call    su_waveshaper
    fld     dword [edx+su_noise_ports.gain]
    fmulp   st1, st0
    ret

%define SU_INCLUDE_WAVESHAPER

%endif

;-------------------------------------------------------------------------------
;   su_op_oscillat function: oscillator, the heart of the synth
;-------------------------------------------------------------------------------
%if OSCILLAT_ID > -1

SECT_TEXT(suoscill)

EXPORT MANGLE_FUNC(su_op_oscillat,0)
    lodsb    ; load the flags
%ifdef INCLUDE_STEREO_OSCILLAT
    jnc     su_op_oscillat_mono
    add     WRK, 4
    call    su_op_oscillat_mono
    fld1                                    ; invert the detune for second run for some stereo separation
    fld     dword [edx+su_osc_ports.detune]
    fsubp   st1
    fstp    dword [edx+su_osc_ports.detune]
    sub     WRK, 4
su_op_oscillat_mono:
%endif
    fld     dword [edx+su_osc_ports.transpose]
    fsub    dword [c_0_5]
    fdiv    dword [c_i128]
    fld     dword [edx+su_osc_ports.detune]
    fsub    dword [c_0_5]
    fadd    st0
    faddp   st1
    test    al, byte LFO
    jnz     su_op_oscillat_skipnote
    fiadd   dword [ecx+su_unit.size-su_voice.workspace+su_voice.note]               ; // st0 is note, st1 is t+d offset
su_op_oscillat_skipnote:
    fmul    dword [c_i12]
    call    MANGLE_FUNC(su_power,0)
    test    al, byte LFO
    jz      short su_op_oscillat_normalize_note
    fmul    dword [c_lfo_normalize]  ; // st0 is now frequency for lfo
    jmp     short su_op_oscillat_normalized
su_op_oscillat_normalize_note:
    fmul    dword [c_freq_normalize]  ; // st0 is now frequency
su_op_oscillat_normalized:
    fadd    dword [WRK+su_osc_wrk.phase]
    fst     dword [WRK+su_osc_wrk.phase]
    fadd    dword [edx+su_osc_ports.phaseofs]
    fld1
    fadd    st1, st0
    fxch
    fprem
    fstp    st1
    fld     dword [edx+su_osc_ports.color]               ; // c      p
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
    ; finally, shape the oscillator and apply gain
    fld     dword [edx+su_osc_ports.shape]
    call    su_waveshaper
su_op_oscillat_gain:
    fld     dword [edx+su_osc_ports.gain]
    fmulp   st1, st0
    ret
    %define SU_INCLUDE_WAVESHAPER

SECT_DATA(suconst)

%ifndef C_FREQ_NORMALIZE
    c_freq_normalize        dd      0.000092696138  ; // 220.0/(2^(69/12)) / 44100.0
    %define C_FREQ_NORMALIZE
%endif
    c_lfo_normalize         dd      0.000038

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
su_oscillat_sine_do
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
    fmul    dword [c_16]                    ; 16*p
    push    eax
    push    eax
    fistp   dword [esp]                     ; s=int(16*p), stack empty
    fld1                                    ; 1
    pop     eax
    and     al, 0xf                         ; ax=int(16*p) & 15, stack: 1
    bt      word [VAL-4],ax                 ; if bit ax of the gate word is set
    jc      go4kVCO_gate_bit                ;   goto gate_bit
    fsub    st0, st0                        ; stack: 0
go4kVCO_gate_bit:                           ; stack: 0/1, let's call it x
    fld     dword [WRK+su_osc_wrk.gatestate] ; g x, g is gatestate, x is the input to this filter 0/1
    fsub    st1                             ; g-x x
    fmul    dword [c_dc_const]              ; c(g-x) x
    faddp   st1, st0                        ; x+c(g-x)
    fst     dword [WRK+su_osc_wrk.gatestate] ; g'=x+c(g-x)
    pop     eax                             ; Another way to see this (c~0.996)
    ret                                     ; g'=cg+(1-c)x
    ; This is a low-pass to smooth the gate transitions

SECT_DATA(suconst)

%ifndef C_16
    c_16                    dd      16.0
    %define C_16
%endif

%ifndef C_DC_CONST
    c_dc_const              dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
    %define C_DC_CONST
%endif

%endif

;-------------------------------------------------------------------------------
;   LOADVAL opcode
;-------------------------------------------------------------------------------
;   Input:      edx     :   pointer to unit ports
; 
;   Mono version: push 2*v-1 on stack, where v is the input to port "value"
;   Stereo version: push 2*v-1 twice on stack
;-------------------------------------------------------------------------------
%if LOADVAL_ID > -1

SECT_TEXT(suloadvl)

EXPORT MANGLE_FUNC(su_op_loadval,0)
%ifdef INCLUDE_STEREO_LOAD_VAL
    jnc     su_op_loadval_mono
    call    su_op_loadval_mono
su_op_loadval_mono:
%endif
    fld     dword [edx+su_load_val_ports.value] ; v
    fsub    dword [c_0_5]                       ; v-.5
    fadd    st0                                 ; 2*v-1
    ret

%endif ; SU_USE_LOAD_VAL


;-------------------------------------------------------------------------------
;   RECEIVE opcode
;-------------------------------------------------------------------------------
;   Mono version:   push l on stack, where l is the left channel received
;   Stereo version: push l r on stack
;-------------------------------------------------------------------------------
%if RECEIVE_ID > -1

SECT_TEXT(sureceiv)

EXPORT MANGLE_FUNC(su_op_receive,0)
    lea     ecx, dword [WRK+su_unit.ports]    
%ifdef INCLUDE_STEREO_RECEIVE
    jnc     su_op_receive_mono
    xor     eax,eax
    fld     dword [ecx+su_receive_ports.right]
    mov     dword [ecx+su_receive_ports.right],eax
su_op_receive_mono:
%else
    xor     eax,eax
%endif
    fld     dword [ecx+su_receive_ports.left]
    mov     dword [ecx+su_receive_ports.left],eax
    ret

%endif ; RECEIVE_ID > -1
