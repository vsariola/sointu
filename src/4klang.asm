%include "4klang.inc"

%define WRK ebp             ; // alias for unit workspace
%define VAL esi             ; // alias for unit values (transformed/untransformed)
%define COM ebx             ; // alias for instrument opcodes

%macro TRANSFORM_VALUES 1
    push %1 %+ .size/4
    call go4kTransformValues
%endmacro

; if SINGLE_FILE is defined, then it means that the whole 4klang.asm will be included
; somewhere else where patterns, pattern_lists and synth_instructions are defined
; Otherwise, they are extern and linker should link them
%ifndef SINGLE_FILE
    extern MANGLE_DATA(go4k_patterns)
    extern MANGLE_DATA(go4k_pattern_lists)
    extern MANGLE_DATA(go4k_synth_instructions)

    %ifdef GO4K_USE_DLL
        ; delay times should also come from the song, but only if the delay is used.
        extern MANGLE_DATA(go4k_delay_times)
    %endif
%endif SINGLE_FILE

; //========================================================================================
; //    .bss section
; //========================================================================================
SECT_BSS(g4kbss1)

; // the one and only synth object
%if MAX_VOICES > 1
go4k_voiceindex         resd    16
%endif
go4k_transformed_values resd    16
go4k_synth_wrk          resb    go4k_synth.size
EXPORT MANGLE_DATA(go4k_delay_buffer_ofs)
                        resd    1
EXPORT MANGLE_DATA(go4k_delay_buffer)
                        resd    16*16*go4kDLL_wrk.size

%ifdef AUTHORING
EXPORT MANGLE_DATA(_4klang_current_tick)
                        resd    0
%endif

%ifdef GO4K_USE_ENVELOPE_RECORDINGS
EXPORT MANGLE_DATA(_4klang_envelope_buffer)
                        resd    ((MAX_SAMPLES)/8) ; // samples every 256 samples and stores 16*2 = 32 values
%endif
%ifdef GO4K_USE_NOTE_RECORDINGS
EXPORT MANGLE_DATA(_4klang_note_buffer)
                        resd    ((MAX_SAMPLES)/8) ; // samples every 256 samples and stores 16*2 = 32 values
%endif

; //========================================================================================
; //    .g4kdat section (initialized data for go4k)
; //========================================================================================
SECT_DATA(g4kdat1)

; // some synth constants
go4k_synth_commands     dd  0
                        dd  MANGLE_FUNC(go4kENV_func,0)
                        dd  MANGLE_FUNC(go4kVCO_func,0)
                        dd  MANGLE_FUNC(go4kVCF_func,0)
                        dd  MANGLE_FUNC(go4kDST_func,0)
                        dd  MANGLE_FUNC(go4kDLL_func,0)
                        dd  MANGLE_FUNC(go4kFOP_func,0)
                        dd  MANGLE_FUNC(go4kFST_func,0)
                        dd  MANGLE_FUNC(go4kPAN_func,0)
                        dd  MANGLE_FUNC(go4kOUT_func,0)
                        dd  MANGLE_FUNC(go4kACC_func,0)
                        dd  MANGLE_FUNC(go4kFLD_func,0)
%ifdef  GO4K_USE_GLITCH
                        dd  MANGLE_FUNC(go4kGLITCH_func,0)
%endif
%ifdef  GO4K_USE_FSTG
                        dd  MANGLE_FUNC(go4kFSTG_func,0)
%endif

SECT_DATA(g4kdat2)

%ifdef GO4K_USE_16BIT_OUTPUT
c_32767                 dd      32767.0
%endif
c_i128                  dd      0.0078125
c_RandDiv               dd      65536*32768
c_0_5                   dd      0.5
%ifdef GO4K_USE_VCO_GATE
c_16                    dd      16.0
%endif
%ifdef GO4K_USE_DLL_CHORUS
DLL_DEPTH               dd      1024.0
%endif
%ifdef GO4K_USE_DLL_DC_FILTER
c_dc_const              dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
%else
    %ifdef GO4K_USE_VCO_GATE
c_dc_const              dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
    %endif
%endif
EXPORT MANGLE_DATA(RandSeed)
                        dd      1
c_24                    dd      24
c_i12                   dd      0x3DAAAAAA
FREQ_NORMALIZE          dd      0.000092696138  ; // 220.0/(2^(69/12)) / 44100.0
EXPORT MANGLE_DATA(LFO_NORMALIZE)
                        dd      DEF_LFO_NORMALIZE
%ifdef GO4K_USE_GROOVE_PATTERN
go4k_groove_pattern     dw      0011100111001110b
%endif

;-------------------------------------------------------------------------------
;   FloatRandomNumber function
;-------------------------------------------------------------------------------
;   Output:     st0     :   result
;-------------------------------------------------------------------------------
SECT_TEXT(crtemui)

EXPORT MANGLE_FUNC(FloatRandomNumber,0)
    push    eax
    imul    eax,dword [MANGLE_DATA(RandSeed)],16007
    mov     dword [MANGLE_DATA(RandSeed)], eax
    fild    dword [MANGLE_DATA(RandSeed)]
    fidiv   dword [c_RandDiv]
    pop     eax
    ret

;-------------------------------------------------------------------------------
;   Waveshaper function
;-------------------------------------------------------------------------------
;   Input:      st0     :   a - the shaping coefficient
;               st1     :   x - input value
;   Output:     st0     :   (1+k)*x/(1+k*abs(x)), where k=2*m/(1-m) and m=2*a-1
;                          and x is clamped first if GO4K_USE_WAVESHAPER_CLIP
;-------------------------------------------------------------------------------
%ifdef INCLUDE_WAVESHAPER

SECT_TEXT(g4kcod2)

go4kWaveshaper:                             ; a x
%ifdef GO4K_USE_WAVESHAPER_CLIP
    fxch                                    ; x a
    fld1                                    ; 1 x a
    fucomi  st1                             ; if (1 <= x)
    jbe     short go4kWaveshaper_clip       ;   goto go4kWaveshaper_clip
    fchs                                    ; -1 x a
    fucomi  st1                             ; if (-1 < x)
    fcmovb  st0, st1                        ;   x x a
go4kWaveshaper_clip:
    fstp    st1                             ; x' a, where x' = clamp(x)
    fxch                                    ; a x' (from now on just called x)
%endif
    fsub    dword [c_0_5]                   ; a-.5 x
    fadd    st0                             ; 2*a-1 x
    fst     dword [esp-4]                   ; m=2*a-1 x
    fadd    st0                             ; 2*m x
    fld1                                    ; 1 2*m x
    fsub    dword [esp-4]                   ; 1-m 2*m x
    fdivp   st1, st0                        ; k=2*m/(1-m) x
    fld     st1                             ; x k x
    fabs                                    ; abs(x) k x
    fmul    st1                             ; k*abs(x) k x
    fld1                                    ; 1 k*abs(x) k x
    faddp   st1, st0                        ; 1+k*abs(x) k x
    fxch    st1                             ; k 1+k*abs(x) x
    fld1                                    ; 1 k 1+k*abs(x) x
    faddp   st1, st0                        ; 1+k 1+k*abs(x) x
    fmulp   st2                             ; 1+k*abs(x) (1+k)*x
    fdivp   st1, st0                        ; (1+k)*x/(1+k*abs(x))
    ret

%endif ; INCLUDE_WAVESHAPER

;-------------------------------------------------------------------------------
;   TransformValues function
;-------------------------------------------------------------------------------
;   Input:      [esp]   :   number of bytes to transform
;               esi     :   pointer to byte stream
;   Output:     eax     :   last transformed byte (zero extended)
;               edx     :   pointer to go4k_transformed_values, containing
;                           each byte transformed as x/128.0f+modulations
;               esi     :   updated to point after the transformed bytes
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcod3)

go4kTransformValues:
    push    ecx
    xor     ecx, ecx
    xor     eax, eax
    mov     edx, go4k_transformed_values
go4kTransformValues_loop:
    lodsb
    push    eax
    fild    dword [esp]
    fmul    dword [c_i128]
    fadd    dword [WRK+MAX_WORK_VARS*4+ecx*4]
    fstp    dword [edx+ecx*4]
    pop     eax
    inc     ecx
    cmp     ecx, dword [esp+8]
    jl      go4kTransformValues_loop
    pop     ecx
    ret     4

;-------------------------------------------------------------------------------
;   ENVMap function
;-------------------------------------------------------------------------------
;   Input:      eax     :   envelope parameter (0 = attac, 1 = decay...)
;               edx     :   pointer to go4k_transformed_values
;   Output:     st0     :   2^(-24*x), where x is the parameter in the range 0-1
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcod4)

go4kENVMap:
    fld     dword [edx+eax*4]   ; x, where x is the parameter in the range 0-1
    fimul   dword [c_24]        ; 24*x
    fchs                        ; -24*x
    ; flow into Power function, which outputs 2^(-24*x)

;-------------------------------------------------------------------------------
;   Power function (2^x)
;-------------------------------------------------------------------------------
;   Input:      st0     :   x
;   Output:     st0     :   2^x
;-------------------------------------------------------------------------------
EXPORT MANGLE_FUNC(Power,0) ; x
    fld1          ; 1 x
    fld st1       ; x 1 x
    fprem         ; mod(x,1) 1 x
    f2xm1         ; 2^mod(x,1)-1 1 x
    faddp st1,st0 ; 2^mod(x,1) x
    fscale        ; 2^mod(x,1)*2^trunc(x) x
                  ; Equal to:
                  ; 2^x x
    fstp st1      ; 2^x
    ret

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
SECT_TEXT(g4kcoda)

EXPORT MANGLE_FUNC(go4kENV_func,0)
    TRANSFORM_VALUES go4kENV_val
%ifdef GO4K_USE_ENV_CHECK
    mov     eax, dword [ecx-4]  ; eax = go4k_instrument.note
    test    eax, eax            ; if (eax != 0) //  if current note still active
    jne     go4kENV_func_do     ;   goto go4kENV_func_do
    fldz
    ret                         ; return 0
%endif
go4kENV_func_do:
    mov     eax, dword [ecx-8]                  ; eax = go4k_instrument.release
    test    eax, eax                            ; if (eax == 0)
    je      go4kENV_func_process                ;   goto process
    mov     dword [WRK+go4kENV_wrk.state], ENV_STATE_RELEASE  ; [state]=RELEASE
go4kENV_func_process:
    mov     eax, dword [WRK+go4kENV_wrk.state]  ; al=[state]
    fld     dword [WRK+go4kENV_wrk.level]       ; x=[level]
    cmp     al, ENV_STATE_SUSTAIN               ; if (al==SUSTAIN)
    je      short go4kENV_func_leave2           ;   goto leave2
go4kENV_func_attac:
    cmp     al, ENV_STATE_ATTAC                 ; if (al!=ATTAC)
    jne     short go4kENV_func_decay            ;   goto decay
    call    go4kENVMap                          ; a x, where a=attack
    faddp   st1, st0                            ; a+x
    fld1                                        ; 1 a+x
    fucomi  st1                                 ; if (a+x<=1) // is attack complete?
    fcmovnb st0, st1                            ;   a+x a+x
    jbe     short go4kENV_func_statechange      ; else goto statechange
go4kENV_func_decay:
    cmp     al, ENV_STATE_DECAY                 ; if (al!=DECAY)
    jne     short go4kENV_func_release          ;   goto release
    call    go4kENVMap                          ; d x, where d=decay
    fsubp   st1, st0                            ; x-d
    fld     dword [edx+go4kENV_val.sustain]     ; s x-d, where s=sustain
    fucomi  st1                                 ; if (x-d>s) // is decay complete?
    fcmovb  st0, st1                            ;   x-d x-d
    jnc     short go4kENV_func_statechange      ; else goto statechange
go4kENV_func_release:
    cmp     al, ENV_STATE_RELEASE               ; if (al!=RELEASE)
    jne     short go4kENV_func_leave            ;   goto leave
    call    go4kENVMap                          ; r x, where r=release
    fsubp   st1, st0                            ; x-r
    fldz                                        ; 0 x-r
    fucomi  st1                                 ; if (x-r>0) // is release complete?
    fcmovb  st0, st1                            ;   x-r x-r, then goto leave
    jc      short go4kENV_func_leave
go4kENV_func_statechange:
    inc     dword [WRK+go4kENV_wrk.state]       ; [state]++
go4kENV_func_leave:
    fstp    st1                                 ; x', where x' is the new value
    fst     dword [WRK+go4kENV_wrk.level]       ; [level]=x'
go4kENV_func_leave2:
    fmul    dword [edx+go4kENV_val.gain]        ; [gain]*x'
    ret

;-------------------------------------------------------------------------------
;   VCO Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;   Output:     st0     :   oscillator value
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodb)

EXPORT MANGLE_FUNC(go4kVCO_func,0)
    TRANSFORM_VALUES go4kVCO_val
%ifdef GO4K_USE_VCO_CHECK
; check if current note still active
    mov     eax, dword [ecx-4]
    test    eax, eax
    jne     go4kVCO_func_do
%ifdef GO4K_USE_VCO_STEREO
    movzx   eax, byte [VAL-1]           ; // get flags and check for stereo
    test    al, byte VCO_STEREO
    jz      short go4kVCO_func_nostereoout
    fldz
go4kVCO_func_nostereoout:
%endif
    fldz
    ret
go4kVCO_func_do:
%endif
    movzx   eax, byte [VAL-1]           ; // get flags
%ifdef GO4K_USE_VCO_STEREO
    test    al, byte VCO_STEREO
    jz      short go4kVCO_func_nopswap
    fld     dword [WRK+go4kVCO_wrk.phase]   ;// swap left/right phase values for first stereo run
    fld     dword [WRK+go4kVCO_wrk.phase2]
    fstp    dword [WRK+go4kVCO_wrk.phase]
    fstp    dword [WRK+go4kVCO_wrk.phase2]
go4kVCO_func_nopswap:
%endif
go4kVCO_func_process:
    fld     dword [edx+go4kVCO_val.transpose]
    fsub    dword [c_0_5]
    fdiv    dword [c_i128]
    fld     dword [edx+go4kVCO_val.detune]
    fsub    dword [c_0_5]
    fadd    st0
%ifdef GO4K_USE_VCO_STEREO
    test    al, byte VCO_STEREO
    jz      short go4kVCO_func_nodswap
    fchs    ;// negate detune for stereo
go4kVCO_func_nodswap:
%endif
    faddp   st1
%ifdef GO4K_USE_VCO_MOD_DM
    fadd    dword [WRK+go4kVCO_wrk.detune_mod]
%endif
    test    al, byte LFO
    jnz     go4kVCO_func_skipnote
    fiadd   dword [ecx-4]               ; // st0 is note, st1 is t+d offset
go4kVCO_func_skipnote:
    fmul    dword [c_i12]
    call    MANGLE_FUNC(Power,0)
    test    al, byte LFO
    jz      short go4kVCO_func_normalize_note
    fmul    dword [_LFO_NORMALIZE]  ; // st0 is now frequency for lfo
    jmp     short go4kVCO_func_normalized
go4kVCO_func_normalize_note:
    fmul    dword [FREQ_NORMALIZE]  ; // st0 is now frequency
go4kVCO_func_normalized:
    fadd    dword [WRK+go4kVCO_wrk.phase]
%ifdef GO4K_USE_VCO_MOD_FM
    fadd    dword [WRK+go4kVCO_wrk.freq_mod]
%endif
    fld1
    fadd    st1, st0
    fxch
    fprem
    fstp    st1
    fst     dword [WRK+go4kVCO_wrk.phase]
%ifdef GO4K_USE_VCO_PHASE_OFFSET
    fadd    dword [edx+go4kVCO_val.phaseofs]
    fld1
    fadd    st1, st0
    fxch
    fprem
    fstp    st1                                         ; // p
%endif
    fld     dword [edx+go4kVCO_val.color]               ; // c      p
go4kVCO_func_sine:
    test    al, byte SINE
    jz      short go4kVCO_func_trisaw
    call    go4kVCO_sine
go4kVCO_func_trisaw:
    test    al, byte TRISAW
    jz      short go4kVCO_func_pulse
    call    go4kVCO_trisaw
go4kVCO_func_pulse:
    test    al, byte PULSE
%ifdef GO4K_USE_VCO_GATE
    jz      short go4kVCO_func_gate
%else
    jz      short go4kVCO_func_noise
%endif
    call    go4kVCO_pulse
%ifdef GO4K_USE_VCO_GATE
go4kVCO_func_gate:
    test    al, byte GATE
    jz      short go4kVCO_func_noise
    call    go4kVCO_gate
%endif
go4kVCO_func_noise:
    test    al, byte NOISE
    jz      short go4kVCO_func_end
    call    MANGLE_FUNC(FloatRandomNumber,0)
    fstp    st1
    fstp    st1
go4kVCO_func_end:
%ifdef GO4K_USE_VCO_SHAPE
    fld     dword [edx+go4kVCO_val.shape]
    call    go4kWaveshaper
%endif
    fld     dword [edx+go4kVCO_val.gain]
    fmulp   st1, st0

%ifdef GO4K_USE_VCO_STEREO
    test    al, byte VCO_STEREO
    jz      short go4kVCO_func_stereodone
    sub     al, byte VCO_STEREO
    fld     dword [WRK+go4kVCO_wrk.phase]   ;// swap left/right phase values again for second stereo run
    fld     dword [WRK+go4kVCO_wrk.phase2]
    fstp    dword [WRK+go4kVCO_wrk.phase]
    fstp    dword [WRK+go4kVCO_wrk.phase2]
    jmp     go4kVCO_func_process
go4kVCO_func_stereodone:
%endif
    ret

SECT_TEXT(g4kcodp)

go4kVCO_pulse:
    fucomi  st1                             ; // c      p
    fld1
    jnc     short go4kVCO_func_pulse_up     ; // +1     c       p
    fchs                                    ; // -1     c       p
go4kVCO_func_pulse_up:
    fstp    st1                             ; // +-1    p
    fstp    st1                             ; // +-1
    ret

SECT_TEXT(g4kcodt)

go4kVCO_trisaw:
    fucomi  st1                             ; // c      p
    jnc     short go4kVCO_func_trisaw_up
    fld1                                    ; // 1      c       p
    fsubr   st2, st0                        ; // 1      c       1-p
    fsubrp  st1, st0                        ; // 1-c    1-p
go4kVCO_func_trisaw_up:
    fdivp   st1, st0                        ; // tp'/tc
    fadd    st0                             ; // 2*''
    fld1                                    ; // 1      2*''
    fsubp   st1, st0                        ; // 2*''-1
    ret

SECT_TEXT(g4kcods)

go4kVCO_sine:
    fucomi  st1                             ; // c      p
    jnc     short go4kVCO_func_sine_do
    fstp    st1
    fsub    st0, st0                        ; // 0
    ret
go4kVCO_func_sine_do
    fdivp   st1, st0                        ; // p/c
    fldpi                                   ; // pi     p
    fadd    st0                             ; // 2*pi   p
    fmulp   st1, st0                        ; // 2*pi*p
    fsin                                    ; // sin(2*pi*p)
    ret

%ifdef GO4K_USE_VCO_GATE

SECT_TEXT(g4kcodq)

go4kVCO_gate:
    fxch                                    ; // p      c
    fstp    st1                             ; // p
    fmul    dword [c_16]                    ; // p'
    push    eax
    push    eax
    fistp   dword [esp]                     ; // -
    fld1                                    ; // 1
    pop     eax
    and     al, 0xf
    bt      word [VAL-5],ax
    jc      go4kVCO_gate_bit
    fsub    st0, st0                        ; // 0
go4kVCO_gate_bit:
    fld     dword [WRK+go4kVCO_wrk.gatestate]       ; // f      x
    fsub    st1                             ; // f-x    x
    fmul    dword [c_dc_const]              ; // c(f-x) x
    faddp   st1, st0                        ; // x'
    fst     dword [WRK+go4kVCO_wrk.gatestate]
    pop     eax
    ret
%endif

;-------------------------------------------------------------------------------
;   VCF Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   signal
;   Output:     st0     :   filtered signal
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodc)

EXPORT MANGLE_FUNC(go4kVCF_func,0)
    TRANSFORM_VALUES go4kVCF_val
%ifdef GO4K_USE_VCF_CHECK
; check if current note still active
    mov     eax, dword [ecx-4]
    test    eax, eax
    jne     go4kVCF_func_do
    ret
go4kVCF_func_do:
%endif
    movzx   eax, byte [VAL-1]               ; // get type flag

    fld     dword [edx+go4kVCF_val.res]     ; //    r       in
    fstp    dword [esp-8]

    fld     dword [edx+go4kVCF_val.freq]    ; //    f       in
    fmul    st0, st0                        ; // square the input so we never get negative and also have a smoother behaviour in the lower frequencies
    fstp    dword [esp-4]                   ; //    in

%ifdef GO4K_USE_VCF_STEREO
    test    al, byte STEREO
    jz      short go4kVCF_func_process
    add     WRK, go4kVCF_wrk.low2
go4kVCF_func_stereoloop:                    ; // switch channels
    fxch    st1                             ; //    inr     inl
%endif

go4kVCF_func_process:
    fld     dword [esp-8]
    fld     dword [esp-4]
    fmul    dword [WRK+go4kVCF_wrk.band]    ; //    f*b     r       in
    fadd    dword [WRK+go4kVCF_wrk.low]     ; //    l'      r       in
    fst     dword [WRK+go4kVCF_wrk.low]     ; //    l'      r       in
    fsubp   st2, st0                        ; //    r       in-l'
    fmul    dword [WRK+go4kVCF_wrk.band]    ; //    r*b     in-l'
    fsubp   st1, st0                        ; //    h'
    fst     dword [WRK+go4kVCF_wrk.high]    ; //    h'
    fmul    dword [esp-4]                   ; //    h'*f
    fadd    dword [WRK+go4kVCF_wrk.band]    ; //    b'
    fstp    dword [WRK+go4kVCF_wrk.band]
    fldz
go4kVCF_func_low:
    test    al, byte LOWPASS
    jz      short go4kVCF_func_high
    fadd    dword [WRK+go4kVCF_wrk.low]
go4kVCF_func_high:
%ifdef GO4K_USE_VCF_HIGH
    test    al, byte HIGHPASS
    jz      short go4kVCF_func_band
    fadd    dword [WRK+go4kVCF_wrk.high]
%endif
go4kVCF_func_band:
%ifdef GO4K_USE_VCF_BAND
    test    al, byte BANDPASS
    jz      short go4kVCF_func_peak
    fadd    dword [WRK+go4kVCF_wrk.band]
%endif
go4kVCF_func_peak:
%ifdef GO4K_USE_VCF_PEAK
    test    al, byte PEAK
    jz      short go4kVCF_func_processdone
    fadd    dword [WRK+go4kVCF_wrk.low]
    fsub    dword [WRK+go4kVCF_wrk.high]
%endif
go4kVCF_func_processdone:

%ifdef GO4K_USE_VCF_STEREO
    test    al, byte STEREO                 ; // outr   inl
    jz      short go4kVCF_func_end
    sub     al, byte STEREO
    sub     WRK, go4kVCF_wrk.low2
    jmp     go4kVCF_func_stereoloop
%endif

go4kVCF_func_end:                           ; // value  -       -       -       -
    ret

;-------------------------------------------------------------------------------
;   DST Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   signal
;   Output:     st0     :   distorted signal
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodd)

EXPORT MANGLE_FUNC(go4kDST_func,0)
%ifdef GO4K_USE_DST
    TRANSFORM_VALUES go4kDST_val
%ifdef GO4K_USE_DST_CHECK
; check if current note still active
    mov     eax, dword [ecx-4]
    test    eax, eax
    jne     go4kDST_func_do
    ret
go4kDST_func_do:
%endif
    movzx   eax, byte [VAL-1]               ; // get type flag
%ifdef  GO4K_USE_DST_SH
    fld     dword [edx+go4kDST_val.snhfreq] ; //    snh     in      (inr)
    fmul    st0, st0                        ; // square the input so we never get negative and also have a smoother behaviour in the lower frequencies
    fchs
    fadd    dword [WRK+go4kDST_wrk.snhphase]; //    snh'    in      (inr)
    fst     dword [WRK+go4kDST_wrk.snhphase]
    fldz                                    ; //    0       snh'    in      (inr)
    fucomip st1                             ; //    snh'    in      (inr)
    fstp    dword [esp-4]                   ; //    in      (inr)
    jc      short go4kDST_func_hold
    fld1                                    ; //    1       in      (inr)
    fadd    dword [esp-4]                   ; //    1+snh'  in      (inr)
    fstp    dword [WRK+go4kDST_wrk.snhphase]; //    in      (inr)
%endif
; // calc pregain and postgain
%ifdef GO4K_USE_DST_STEREO
    test    al, byte STEREO
    jz      short go4kDST_func_mono
    fxch    st1                             ; //    inr     inl
    fld     dword [edx+go4kDST_val.drive]   ; //    drive   inr     inl
    call    go4kWaveshaper                  ; //    outr    inl
%ifdef  GO4K_USE_DST_SH
    fst     dword [WRK+go4kDST_wrk.out2]    ; //    outr    inl
%endif
    fxch    st1                             ; //    inl     outr
go4kDST_func_mono:
%endif
    fld     dword [edx+go4kDST_val.drive]   ; //    drive   in      (outr)
    call    go4kWaveshaper                  ; //    out     (outr)
%ifdef  GO4K_USE_DST_SH
    fst     dword [WRK+go4kDST_wrk.out]     ; //    out'    (outr)
%endif
    ret                                     ; //    out'    (outr)
%ifdef  GO4K_USE_DST_SH
go4kDST_func_hold:                          ; //    in      (inr)
    fstp    st0                             ; //    (inr)
%ifdef GO4K_USE_DST_STEREO
    test    al, byte STEREO
    jz      short go4kDST_func_monohold     ; //    (inr)
    fstp    st0                             ; //
    fld     dword [WRK+go4kDST_wrk.out2]    ; //    outr
go4kDST_func_monohold:
%endif
    fld     dword [WRK+go4kDST_wrk.out]     ; //    out     (outr)
    ret
%endif

%endif

;-------------------------------------------------------------------------------
;   DLL Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   signal
;   Output:     st0     :   delayed signal
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodf)

EXPORT MANGLE_FUNC(go4kDLL_func,0)
%ifdef GO4K_USE_DLL
    TRANSFORM_VALUES go4kDLL_val
    pushad
    movzx   ebx, byte [VAL-(go4kDLL_val.size-go4kDLL_val.delay)/4]  ;// delay length index
%ifdef GO4K_USE_DLL_NOTE_SYNC
    test    ebx, ebx
    jne     go4kDLL_func_process
    fld1
    fild    dword [ecx-4]           ; // load note freq
    fmul    dword [c_i12]
    call    MANGLE_FUNC(Power,0)
    fmul    dword [FREQ_NORMALIZE]  ; // normalize
    fdivp   st1, st0                ; // invert to get numer of samples
    fistp   word [MANGLE_DATA(go4k_delay_times)+ebx*2]  ; store current comb size
%endif
go4kDLL_func_process:
    mov     ecx, eax                            ;// ecx is delay counter
    mov     WRK, dword [MANGLE_DATA(go4k_delay_buffer_ofs)] ;// ebp is current delay
    fld     st0                                 ;// in      in
    fmul    dword [edx+go4kDLL_val.dry]         ;// out     in
    fxch                                        ;// in      out
    fmul    dword [edx+go4kDLL_val.pregain]     ;// in'     out
    fmul    dword [edx+go4kDLL_val.pregain]     ;// in'     out

%ifdef GO4K_USE_DLL_CHORUS
;// update saw lfo for chorus/flanger
    fld     dword [edx+go4kDLL_val.freq]        ;// f       in'     out
    fmul    st0, st0
    fmul    st0, st0
    fdiv    dword [DLL_DEPTH]
    fadd    dword [WRK+go4kDLL_wrk.phase]       ;// p'      in'     out
;// clamp phase to 0,1 (only in editor, since delay can be active quite long)
%ifdef GO4K_USE_DLL_CHORUS_CLAMP
    fld1                                        ;// 1       p'      in'     out
    fadd    st1, st0                            ;// 1       1+p'    in'     out
    fxch                                        ;// 1+p'    1       in'     out
    fprem                                       ;// p''     1       in'     out
    fstp    st1                                 ;// p''     in'     out
%endif
    fst     dword [WRK+go4kDLL_wrk.phase]
;// get current sine value
    fldpi                                       ; // pi     p       in'     out
    fadd    st0                                 ; // 2*pi   p       in'     out
    fmulp   st1, st0                            ; // 2*pi*p in'     out
    fsin                                        ; // sin    in'     out
    fld1                                        ; // 1      sin     in'     out
    faddp   st1, st0                            ; // 1+sin  in'     out
;// mul with depth and convert to samples
    fld     dword [edx+go4kDLL_val.depth]       ; // d      1+sin   in'     out
    fmul    st0, st0
    fmul    st0, st0
    fmul    dword [DLL_DEPTH]
    fmulp   st1, st0
    fistp   dword [esp-4]                       ; // in'    out
%endif

go4kDLL_func_loop:
    movzx   esi, word [MANGLE_DATA(go4k_delay_times)+ebx*2] ; fetch comb size
    mov     eax, dword [WRK+go4kDLL_wrk.index]  ;// eax is current comb index

%ifdef GO4K_USE_DLL_CHORUS
    ;// add lfo offset and wrap buffer
    add     eax, dword [esp-4]
    cmp     eax, esi
    jl      short go4kDLL_func_buffer_nowrap1
    sub     eax, esi
go4kDLL_func_buffer_nowrap1:
%endif

    fld     dword [WRK+eax*4+go4kDLL_wrk.buffer];// cout        in'         out

%ifdef GO4K_USE_DLL_CHORUS
    mov     eax, dword [WRK+go4kDLL_wrk.index]
%endif

    ;// add comb output to current output
    fadd    st2, st0                            ;// cout        in'         out'
%ifdef GO4K_USE_DLL_DAMP
    fld1                                        ;// 1           cout        in'         out'
    fsub    dword [edx+go4kDLL_val.damp]        ;// 1-damp      cout        in'         out'
    fmulp   st1, st0                            ;// cout*d2     in'         out'
    fld     dword [edx+go4kDLL_val.damp]        ;// d1          cout*d2     in'         out'
    fmul    dword [WRK+go4kDLL_wrk.store]       ;// store*d1    cout*d2     in'         out'
    faddp   st1, st0                            ;// store'      in'         out'
    fst     dword [WRK+go4kDLL_wrk.store]       ;// store'      in'         out'
%endif
    fmul    dword [edx+go4kDLL_val.feedback]    ;// cout*fb     in'         out'
%ifdef GO4K_USE_DLL_DC_FILTER
    fadd    st0, st1                            ;// store       in'         out'
    fstp    dword [WRK+eax*4+go4kDLL_wrk.buffer];// in'         out'
%else
    fsub    st0, st1                            ;// store       in'         out'
    fstp    dword [WRK+eax*4+go4kDLL_wrk.buffer];// in'         out'
    fneg
%endif
    ;// wrap comb buffer pos
    inc     eax
    cmp     eax, esi
    jl      short go4kDLL_func_buffer_nowrap2
%ifdef GO4K_USE_DLL_CHORUS
    sub     eax, esi
%else
    xor     eax, eax
%endif
go4kDLL_func_buffer_nowrap2:
    mov     dword [WRK+go4kDLL_wrk.index], eax
    ;// increment buffer pointer to next buffer
    inc     ebx                                 ;// go to next delay length index
    add     WRK, go4kDLL_wrk.size               ;// go to next delay
    mov     dword [MANGLE_DATA(go4k_delay_buffer_ofs)], WRK ;// store next delay offset
    loopne  go4kDLL_func_loop
    fstp    st0                                 ;// out'
    ;// process a dc filter to prevent heavy offsets in reverb
%ifdef GO4K_USE_DLL_DC_FILTER
;   y(n) = x(n) - x(n-1) + R * y(n-1)
    sub     WRK, go4kDLL_wrk.size
    fld     dword [WRK+go4kDLL_wrk.dcout]       ;// dco         out'
    fmul    dword [c_dc_const]                  ;// dcc*dco     out'
    fsub    dword [WRK+go4kDLL_wrk.dcin]        ;// dcc*dco-dci out'
    fxch                                        ;// out'        dcc*dco-dci
    fst     dword [WRK+go4kDLL_wrk.dcin]        ;// out'        dcc*dco-dci
    faddp   st1                                 ;// out'
%ifdef GO4K_USE_UNDENORMALIZE
    fadd    dword [c_0_5]                       ;// add and sub small offset to prevent denormalization
    fsub    dword [c_0_5]
%endif
    fst     dword [WRK+go4kDLL_wrk.dcout]
%endif
    popad
    ret
%endif

;-------------------------------------------------------------------------------
;   GLITCH Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               ?
;   Output:     ?
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodu)

EXPORT MANGLE_FUNC(go4kGLITCH_func,0)
%ifdef GO4K_USE_GLITCH
    TRANSFORM_VALUES go4kGLITCH_val
    pushad

    mov     edi, WRK
    mov     WRK, dword [MANGLE_DATA(go4k_delay_buffer_ofs)] ;// ebp is current delay

;   mov     eax, dword [edx+go4kGLITCH_val.active]
;   or      eax, dword [edi+go4kGLITCH_wrk2.am]
;   test    eax, eax
;   je      go4kGLITCH_func_notactive           ;// out

    fld     dword [edx+go4kGLITCH_val.active]   ;// a       in
    ; // check for activity
    fldz                                        ;// 0       a'      in
    fucomip st1                                 ;// a'      in
    fstp    st0                                 ;// in
    jnc     go4kGLITCH_func_notactive       ;// out

    ;// check for first call and init if so init (using slizesize slot)
    mov     eax, dword [WRK+go4kGLITCH_wrk.slizesize]
    and     eax, eax
    jnz     go4kGLITCH_func_process
        mov     dword [WRK+go4kGLITCH_wrk.index], eax
        mov     dword [WRK+go4kGLITCH_wrk.store], eax
        movzx   ebx, byte [VAL-(go4kGLITCH_val.size-go4kGLITCH_val.slicesize)/4]    ;// slicesize index
        movzx   eax, word [MANGLE_DATA(go4k_delay_times)+ebx*2]                                 ;// fetch slicesize
        push    eax
        fld1
        fild    dword [esp]
        fstp    dword [WRK+go4kGLITCH_wrk.slizesize]
        fstp    dword [WRK+go4kGLITCH_wrk.slicepitch]
        pop     eax
go4kGLITCH_func_process:
    ;// fill buffer until full
    mov     eax, dword [WRK+go4kGLITCH_wrk.store]
    cmp     eax, MAX_DELAY
    jae     go4kGLITCH_func_filldone
        fst     dword [WRK+eax*4+go4kDLL_wrk.buffer]    ;// in
        inc     dword [WRK+go4kGLITCH_wrk.store]
go4kGLITCH_func_filldone:
    ;// save input
    push    eax
    fstp    dword [esp]                                 ;// -

    ;// read from buffer
    push    eax
    fld     dword [WRK+go4kGLITCH_wrk.index]            ;// idx
    fist    dword [esp]
    pop     eax
    fld     dword [WRK+eax*4+go4kDLL_wrk.buffer]        ;// out     idx
    fxch                                                ;// idx     out
    ;// progress readindex with current play speed
    fadd    dword [WRK+go4kGLITCH_wrk.slicepitch]       ;// idx'    out
    fst     dword [WRK+go4kGLITCH_wrk.index]

    ;// check for slice done
    fld     dword [WRK+go4kGLITCH_wrk.slizesize]        ;// size    idx'    out
    fxch                                                ;// idx'    size    out
    fucomip st1                                         ;// idx'    out
    fstp    st0                                         ;// out
    jc  go4kGLITCH_func_process_done
        ;// reinit for next loop
        xor     eax, eax
        mov     dword [WRK+go4kGLITCH_wrk.index], eax

        fld     dword [edx+go4kGLITCH_val.dsize]
        fsub    dword [c_0_5]
        fmul    dword [c_0_5]
        call MANGLE_FUNC(Power,0)
        fmul    dword [WRK+go4kGLITCH_wrk.slizesize]
        fstp    dword [WRK+go4kGLITCH_wrk.slizesize]

        fld     dword [edx+go4kGLITCH_val.dpitch]
        fsub    dword [c_0_5]
        fmul    dword [c_0_5]
        call MANGLE_FUNC(Power,0)
        fmul    dword [WRK+go4kGLITCH_wrk.slicepitch]
        fstp    dword [WRK+go4kGLITCH_wrk.slicepitch]
go4kGLITCH_func_process_done:

    ;// dry wet mix
    fld     dword [edx+go4kGLITCH_val.dry]              ;// dry     out
    fld1                                                ;// 1       dry'    out
    fsub    st1                                         ;// 1-dry'  dry'    out
    fmulp   st2                                         ;// dry'    out'
    fmul    dword [esp]                                 ;// in'     out'
    faddp   st1, st0                                    ;// out'

    pop     eax
    jmp     go4kGLITCH_func_leave
go4kGLITCH_func_notactive:
    ;// mark as uninitialized again (using slizesize slot)
    xor     eax,eax
    mov     dword [WRK+go4kGLITCH_wrk.slizesize], eax
go4kGLITCH_func_leave:
    add     WRK, go4kDLL_wrk.size               ;// go to next delay
    mov     dword [MANGLE_DATA(go4k_delay_buffer_ofs)], WRK ;// store next delay offset
    popad
    ret
%endif

;-------------------------------------------------------------------------------
;   FOP Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               stX     :   depends on the operation
;   Output:     stX     :   depends on the operation
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodg)

EXPORT MANGLE_FUNC(go4kFOP_func,0)
    TRANSFORM_VALUES go4kFOP_val
go4kFOP_func_pop:
    dec     eax
    jnz     go4kFOP_func_addp
    fstp    st0
    ret
go4kFOP_func_addp:
    dec     eax
    jnz     go4kFOP_func_mulp
    faddp   st1, st0
    ret
go4kFOP_func_mulp:
    dec     eax
    jnz     go4kFOP_func_push
    fmulp   st1, st0
    ret
go4kFOP_func_push:
    dec     eax
    jnz     go4kFOP_func_xch
    fld     st0
    ret
go4kFOP_func_xch:
    dec     eax
    jnz     go4kFOP_func_add
    fxch
    ret
go4kFOP_func_add:
    dec     eax
    jnz     go4kFOP_func_mul
    fadd    st1
    ret
go4kFOP_func_mul:
    dec     eax
    jnz     go4kFOP_func_addp2
    fmul    st1
    ret
go4kFOP_func_addp2:
    dec     eax
    jnz     go4kFOP_func_loadnote
    faddp   st2, st0
    faddp   st2, st0
    ret
go4kFOP_func_loadnote:
    dec     eax
    jnz     go4kFOP_func_mulp2
    fild    dword [ecx-4]
    fmul    dword [c_i128]
    ret
go4kFOP_func_mulp2:
    fmulp   st2, st0
    fmulp   st2, st0
    ret

;-------------------------------------------------------------------------------
;   FST Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   signal
;   Output:     (st0)   :   signal, unless popped
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodh)

EXPORT MANGLE_FUNC(go4kFST_func,0)
    TRANSFORM_VALUES go4kFST_val
    fld     dword [edx+go4kFST_val.amount]
    fsub    dword [c_0_5]
    fadd    st0
    fmul    st1
    lodsw
    and     eax, 0x00003fff                 ; // eax is destination slot
    test    word [VAL-2], FST_ADD
    jz      go4kFST_func_set
    fadd    dword [ecx+eax*4]
go4kFST_func_set:
    fstp    dword [ecx+eax*4]
    test    word [VAL-2], FST_POP
    jz      go4kFST_func_done
    fstp    st0
go4kFST_func_done:
    ret

;-------------------------------------------------------------------------------
;   FLD Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;   Output:     st0     :   2*v-1, where v is the loaded value
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodm)

EXPORT MANGLE_FUNC(go4kFLD_func,0)
%ifdef GO4K_USE_FLD
    TRANSFORM_VALUES go4kFLD_val
    fld     dword [edx+go4kFLD_val.value]   ; v
    fsub    dword [c_0_5]                   ; v-.5
    fadd    st0                             ; 2*v-1
%endif
    ret

;-------------------------------------------------------------------------------
;   FSTG Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   signal
;   Output:     (st0)   :   signal, unless popped
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
%ifdef GO4K_USE_FSTG

SECT_TEXT(g4kcodi)

EXPORT MANGLE_FUNC(go4kFSTG_func,0)
    TRANSFORM_VALUES go4kFSTG_val
%ifdef GO4K_USE_FSTG_CHECK
; check if current note still active
    mov     eax, dword [ecx-4]
    test    eax, eax
    jne     go4kFSTG_func_do
    lodsw
    jmp     go4kFSTG_func_testpop
go4kFSTG_func_do:
%endif
    fld     dword [edx+go4kFST_val.amount]
    fsub    dword [c_0_5]
    fadd    st0
    fmul    st1
    lodsw
    and     eax, 0x00003fff                 ; // eax is destination slot
    test    word [VAL-2], FST_ADD
    jz      go4kFSTG_func_set
    fadd    dword [go4k_synth_wrk+eax*4]
go4kFSTG_func_set:
%if MAX_VOICES > 1
    fst     dword [go4k_synth_wrk+eax*4]
    fstp    dword [go4k_synth_wrk+eax*4+go4k_instrument.size]
%else
    fstp    dword [go4k_synth_wrk+eax*4]
%endif
go4kFSTG_func_testpop:
    test    word [VAL-2], FST_POP
    jz      go4kFSTG_func_done
    fstp    st0
go4kFSTG_func_done:
    ret
%endif


;-------------------------------------------------------------------------------
;   PAN Tick
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   s, the signal
;   Output:     st0     :   s*(1-p), where p is the panning in [0,1] range
;               st1     :   s*p
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodj)

EXPORT MANGLE_FUNC(go4kPAN_func,0)
%ifdef GO4K_USE_PAN
    TRANSFORM_VALUES go4kPAN_val
    fld     dword [edx+go4kPAN_val.panning]     ; p s
    fmul    st1                                 ; p*s s
    fsub    st1, st0                            ; p*s s-p*s
                                                ; Equal to
                                                ; s*p s*(1-p)
    fxch                                        ; s*(1-p) s*p
%else
    fmul    dword [c_0_5]                       ; s*.5
    fld     st0                                 ; s*.5 s*.5
%endif
    ret


;-------------------------------------------------------------------------------
;   OUT Tick (stores stereo signal pair in temp buffers of the instrument)
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   l, the left signal
;               st1     :   r, the right signal
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodk)

EXPORT MANGLE_FUNC(go4kOUT_func,0)                              ;// l       r
    TRANSFORM_VALUES go4kOUT_val
%ifdef  GO4K_USE_GLOBAL_DLL
    pushad
    lea     edi, [ecx+MAX_UNITS*MAX_UNIT_SLOTS*4]
    fld     st1                                         ;// r       l       r
    fld     st1                                         ;// l       r       l       r
    fld     dword [edx+go4kOUT_val.auxsend]             ;// as      l       r       l       r
    fmulp   st1, st0                                    ;// l'      r       l       r
    fstp    dword [edi]                                 ;// r       l       r
    scasd
    fld     dword [edx+go4kOUT_val.auxsend]             ;// as      r       l       r
    fmulp   st1, st0                                    ;// r'      l       r
    fstp    dword [edi]                                 ;// l       r
    scasd
    fld     dword [edx+go4kOUT_val.gain]                ;// g       l       r
    fmulp   st1, st0                                    ;// l'      r
    fstp    dword [edi]                                 ;// r
    scasd
    fld     dword [edx+go4kOUT_val.gain]                ;// g       r
    fmulp   st1, st0                                    ;// r'
    fstp    dword [edi]                                 ;// -
    scasd
    popad
%else
    fld     dword [edx+go4kOUT_val.gain]                ;// g       l       r
    fmulp   st1, st0                                    ;// l'      r
    fstp    dword [ecx+MAX_UNITS*MAX_UNIT_SLOTS*4+8]                            ;// r
    fld     dword [edx+go4kOUT_val.gain]                ;// g       r
    fmulp   st1, st0                                    ;// r'
    fstp    dword [ecx+MAX_UNITS*MAX_UNIT_SLOTS*4+12]                           ;// -

%endif
    ret


;-------------------------------------------------------------------------------
;   ACC Tick (stereo signal accumulation)
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;   Dirty:      eax,edx
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodl)

EXPORT MANGLE_FUNC(go4kACC_func,0)
    TRANSFORM_VALUES go4kACC_val
    pushad
    mov     edi, go4k_synth_wrk
    add     edi, go4k_instrument.size
    sub     edi, eax                    ; // eax already contains the accumulation offset from the go4kTransformValues call
    mov     cl, MAX_INSTRUMENTS*MAX_VOICES
    fldz                                ;// 0
    fldz                                ;// 0       0
go4kACC_func_loop:
    fadd    dword [edi-8]               ;// l       0
    fxch                                ;// 0       l
    fadd    dword [edi-4]               ;// r       l
    fxch                                ;// l       r
    add     edi, go4k_instrument.size
    dec     cl
    jnz     go4kACC_func_loop
    popad
    ret

;-------------------------------------------------------------------------------
;   Update Instrument (allocate voices, set voice to release)
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodw)

go4kUpdateInstrument:
; // get new note
    mov     eax, dword [esp+4+4]                    ; // eax = current tick
    shr     eax, PATTERN_SIZE_SHIFT                 ; // eax = current pattern
    imul    edx, ecx, dword MAX_PATTERNS                ; // edx = instrument pattern list index
    movzx   edx, byte [edx+eax+MANGLE_DATA(go4k_pattern_lists)] ; // edx = pattern index
    mov     eax, dword [esp+4+4]                    ; // eax = current tick
    shl     edx, PATTERN_SIZE_SHIFT
    and     eax, PATTERN_SIZE-1
    movzx   edx, byte [edx+eax+MANGLE_DATA(go4k_patterns)]  ; // edx = requested note in new patterntick
; // apply note changes
    cmp     dl, HLD                                 ; // anything but hold causes action
    je      short go4kUpdateInstrument_done
    inc     dword [edi]                             ; // set release flag if needed
%if MAX_VOICES > 1
    inc     dword [edi+go4k_instrument.size]        ; // set release flag if needed
%endif
    cmp     dl, HLD                                 ; // check for new note
    jl      short go4kUpdateInstrument_done
%if MAX_VOICES > 1
    pushad
    xchg    eax, dword [go4k_voiceindex + ecx*4]
    test    eax, eax
    je      go4kUpdateInstrument_newNote
    add     edi, go4k_instrument.size
go4kUpdateInstrument_newNote:
    xor     al,1
    xchg    dword [go4k_voiceindex + ecx*4], eax
%endif
    pushad
    xor     eax, eax
    mov     ecx, (8+MAX_UNITS*MAX_UNIT_SLOTS*4)/4   ; // clear only relase, note and workspace
    rep     stosd
    popad
    mov     dword [edi+4], edx                      ; // set requested note as current note
%if MAX_VOICES > 1
    popad
%endif
go4kUpdateInstrument_done:
    ret

;-------------------------------------------------------------------------------
;   Render Voices
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodx)

go4kRenderVoices:
    push    ecx                             ; // save current instrument counter
%if MAX_VOICES > 1
    push    COM                             ; // save current instrument command index
    push    VAL                             ; // save current instrument values index
%endif
    call    go4k_VM_process                 ; //  call synth vm for instrument voices
    mov     eax, dword [ecx+go4kENV_wrk.state]
    cmp     al, byte ENV_STATE_OFF
    jne     go4kRenderVoices_next
    xor     eax, eax
    mov     dword [ecx-4], eax              ; // kill note if voice is done
go4kRenderVoices_next:
%if MAX_VOICES > 1
    pop     VAL                             ; // restore instrument value index
    pop     COM                             ; // restore instrument command index
%endif

%ifdef GO4K_USE_BUFFER_RECORDINGS
    mov     eax, dword [esp+16]             ; // get current tick
    shr     eax, 8                          ; // every 256th sample = ~ 172 hz
    shl     eax, 5                          ; // for 16 instruments a 2 voices
    add     eax, dword [esp]
    add     eax, dword [esp]                ; // + 2*currentinstrument+0
%ifdef GO4K_USE_ENVELOPE_RECORDINGS
    mov     edx, dword [ecx+go4kENV_wrk.level]
    mov     dword [MANGLE_DATA(_4klang_envelope_buffer)+eax*4], edx
%endif
%ifdef GO4K_USE_NOTE_RECORDINGS
    mov     edx, dword [ecx-4]
    mov     dword [MANGLE_DATA(_4klang_note_buffer)+eax*4], edx
%endif
%endif

%if MAX_VOICES > 1
    call    go4k_VM_process                 ; //  call synth vm for instrument voices
    mov     eax, dword [ecx+go4kENV_wrk.state]
    cmp     al, byte ENV_STATE_OFF
    jne     go4k_render_instrument_next2
    xor     eax, eax
    mov     dword [ecx-4], eax              ; // kill note if voice is done
go4k_render_instrument_next2:

%ifdef GO4K_USE_BUFFER_RECORDINGS
    mov     eax, dword [esp+16]             ; // get current tick
    shr     eax, 8                          ; // every 256th sample = ~ 172 hz
    shl     eax, 5                          ; // for 16 instruments a 2 voices
    add     eax, dword [esp]
    add     eax, dword [esp]                ; // + 2*currentinstrument+0
%ifdef GO4K_USE_ENVELOPE_RECORDINGS
    mov     edx, dword [ecx+go4kENV_wrk.level]
    mov     dword [MANGLE_DATA(_4klang_envelope_buffer)+eax*4+4], edx
%endif
%ifdef GO4K_USE_NOTE_RECORDINGS
    mov     edx, dword [ecx-4]
    mov     dword [MANGLE_DATA(_4klang_note_buffer)+eax*4+4], edx
%endif
%endif

%endif
    pop     ecx                             ; // restore instrument counter
    ret

;-------------------------------------------------------------------------------
;   _4klang_render function: the entry point for the synth
;-------------------------------------------------------------------------------
;   Has the signature _4klang_render(void *ptr), where ptr is a pointer to
;   the output buffer
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcody)

EXPORT MANGLE_FUNC(_4klang_render,4)
    pushad
    xor     ecx, ecx
%ifdef GO4K_USE_BUFFER_RECORDINGS
    push    ecx
%endif
; loop all ticks
go4k_render_tickloop:
    push    ecx
    xor     ecx, ecx
; loop all samples per tick
go4k_render_sampleloop:
        push    ecx
        xor     ecx, ecx
        mov     ebx, MANGLE_DATA(go4k_synth_instructions) ; // ebx = instrument command index
        mov     VAL, MANGLE_DATA(go4k_synth_parameter_values); // VAL = instrument values index
        mov     edi, _go4k_delay_buffer         ; // get offset of first delay buffer
        mov     dword [MANGLE_DATA(go4k_delay_buffer_ofs)], edi ; // store offset in delaybuffer offset variable
        mov     edi, go4k_synth_wrk             ; // edi = first instrument
; loop all instruments
go4k_render_instrumentloop:
            mov     eax, dword [esp]                ; // eax = current tick sample
            and     eax, eax
            jnz     go4k_render_instrument_process  ; // tick change? (first sample in current tick)
            call    go4kUpdateInstrument            ; // update instrument state
; process instrument
go4k_render_instrument_process:
            call    go4kRenderVoices
            inc     ecx
            cmp     cl, byte MAX_INSTRUMENTS
            jl      go4k_render_instrumentloop
        mov     dword [edi+4], ecx      ; // move a value != 0 into note slot, so processing will be done
        call    go4k_VM_process         ; //  call synth vm for synth
go4k_render_output_sample:
%ifdef GO4K_USE_BUFFER_RECORDINGS
        inc     dword [esp+8]
        xchg    esi, dword [esp+48]     ; // edx = destbuffer
%else
        xchg    esi, dword [esp+44]     ; // edx = destbuffer
%endif
%ifdef  GO4K_CLIP_OUTPUT
        fld     dword [edi-8]
        fld1                                    ; //    1       val
        fucomi  st1                             ; //    1       val
        jbe     short go4k_render_clip1
        fchs                                    ; //    -1      val
        fucomi  st1                             ; //    -1      val
        fcmovb  st0, st1                        ; //    val     -1      (if val > -1)
go4k_render_clip1:
        fstp    st1                             ; //    newval
%ifdef GO4K_USE_16BIT_OUTPUT
        push    eax
        fmul    dword [c_32767]
        fistp   dword [esp]
        pop     eax
        mov     word [esi],ax   ; // store integer converted left sample
        lodsw
%else
        fstp    dword [esi]     ; // store left sample
        lodsd
%endif
        fld     dword [edi-4]
        fld1                                    ; //    1       val
        fucomi  st1                             ; //    1       val
        jbe     short go4k_render_clip2
        fchs                                    ; //    -1      val
        fucomi  st1                             ; //    -1      val
        fcmovb  st0, st1                        ; //    val     -1      (if val > -1)
go4k_render_clip2:
        fstp    st1                             ; //    newval
%ifdef GO4K_USE_16BIT_OUTPUT
        push    eax
        fmul    dword [c_32767]
        fistp   dword [esp]
        pop     eax
        mov     word [esi],ax   ; // store integer converted right sample
        lodsw
%else
        fstp    dword [esi]     ; // store right sample
        lodsd
%endif
%else
        fld     dword [edi-8]
%ifdef GO4K_USE_16BIT_OUTPUT
        push    eax
        fmul    dword [c_32767]
        fistp   dword [esp]
        pop     eax
        mov     word [esi],ax   ; // store integer converted left sample
        lodsw
%else
        fstp    dword [esi]     ; // store left sample
        lodsd
%endif
        fld     dword [edi-4]
%ifdef GO4K_USE_16BIT_OUTPUT
        push    eax
        fmul    dword [c_32767]
        fistp   dword [esp]
        pop     eax
        mov     word [esi],ax   ; // store integer converted right sample
        lodsw
%else
        fstp    dword [esi]     ; // store right sample
        lodsd
%endif
%endif
%ifdef GO4K_USE_BUFFER_RECORDINGS
        xchg    esi, dword [esp+48]
%else
        xchg    esi, dword [esp+44]
%endif
        pop     ecx
        inc     ecx
%ifdef GO4K_USE_GROOVE_PATTERN
        mov     ebx, dword SAMPLES_PER_TICK
        mov     eax, dword [esp]
        and     eax, 0x0f
        bt      dword [go4k_groove_pattern],eax
        jnc     go4k_render_nogroove
        sub     ebx, dword 3000
go4k_render_nogroove:
        cmp     ecx, ebx
%else
        cmp     ecx, dword SAMPLES_PER_TICK
%endif
        jl      go4k_render_sampleloop
    pop     ecx
    inc     ecx
%ifdef AUTHORING
    mov     dword[MANGLE_DATA(_4klang_current_tick)], ecx
%endif
    cmp     ecx, dword MAX_TICKS
    jl      go4k_render_tickloop
%ifdef GO4K_USE_BUFFER_RECORDINGS
    pop     ecx
%endif
    popad
    ret     4

;-------------------------------------------------------------------------------
;   VM_process function (the virtual machine behind the synth)
;-------------------------------------------------------------------------------
;   Input:      edi     :   pointer to the instrument structure
;               VAL     :   pointer to unit values as bytes
;               ebx     :   pointer to instrument instructions
;-------------------------------------------------------------------------------
SECT_TEXT(g4kcodz)

go4k_VM_process:
    lea     WRK, [edi+8]                        ; WRK = workspace start
    mov     ecx, WRK                            ; ecx = workspace start
go4k_VM_process_loop:
    movzx   eax, byte [ebx]                     ; eax = command byte
    inc     ebx
    test    eax, eax                            ; if (eax == 0)
    je      go4k_VM_process_done                ;   goto done
    call    dword [eax*4+go4k_synth_commands]   ; call the function corresponding to command
    add     WRK, MAX_UNIT_SLOTS*4               ; move WRK to next workspace
    jmp     short go4k_VM_process_loop
go4k_VM_process_done:
    add     edi, go4k_instrument.size           ; move edi to next instrument
    ret