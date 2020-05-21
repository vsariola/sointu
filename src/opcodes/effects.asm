;-------------------------------------------------------------------------------
;   DISTORT Tick
;-------------------------------------------------------------------------------
;   Input:      st0     :   x - input value
;   Output:     st0     :  x*a/(1-a+(2*a-1)*abs(x))
;                          where x is clamped first
;-------------------------------------------------------------------------------
%if DISTORT_ID > -1

SECT_TEXT(sudistrt)

EXPORT MANGLE_FUNC(su_op_distort,0)
    %ifdef INCLUDE_STEREO_DISTORT    
        call su_effects_stereohelper
        %define INCLUDE_EFFECTS_STEREOHELPER
    %endif
    fld     dword [INP+su_distort_ports.drive]
    %define SU_INCLUDE_WAVESHAPER
    ; flow into waveshaper
%endif

%ifdef SU_INCLUDE_WAVESHAPER
su_waveshaper:
    fxch                                    ; x a
    call    su_clip
    fxch                                    ; a x' (from now on just called x)
    fld     st0                             ; a a x
    apply fsub dword,c_0_5                  ; a-.5 a x
    fadd    st0                             ; 2*a-1 a x
    fld     st2                             ; x 2*a-1 a x
    fabs                                    ; abs(x) 2*a-1 a x
    fmulp   st1                             ; (2*a-1)*abs(x) a x
    fld1                                    ; 1 (2*a-1)*abs(x) a x
    faddp   st1                             ; 1+(2*a-1)*abs(x) a x
    fsub    st1                             ; 1-a+(2*a-1)*abs(x) a x
    fdivp   st1, st0                        ; a/(1-a+(2*a-1)*abs(x)) x
    fmulp   st1                             ; x*a/(1-a+(2*a-1)*abs(x))
    ret

%define SU_INCLUDE_CLIP

%endif ; SU_USE_DST

;-------------------------------------------------------------------------------
;   HOLD Tick
;-------------------------------------------------------------------------------
%if HOLD_ID > -1

SECT_TEXT(suhold)

EXPORT MANGLE_FUNC(su_op_hold,0)
    %ifdef INCLUDE_STEREO_HOLD   
        call    su_effects_stereohelper
        %define INCLUDE_EFFECTS_STEREOHELPER
    %endif
    fld     dword [INP+su_hold_ports.freq]    ; f x
    fmul    st0, st0                        ; f^2 x
    fchs                                    ; -f^2 x
    fadd    dword [WRK+su_hold_wrk.phase]   ; p-f^2 x
    fst     dword [WRK+su_hold_wrk.phase]   ; p <- p-f^2
    fldz                                    ; 0 p x
    fucomip st1                             ; p x
    fstp    dword [_SP-4]                   ; t=p, x
    jc      short su_op_hold_holding        ; if (0 < p) goto holding
    fld1                                    ; 1 x
    fadd    dword [_SP-4]                   ; 1+t x
    fstp    dword [WRK+su_hold_wrk.phase]   ; x
    fst     dword [WRK+su_hold_wrk.holdval] ; save holded value
    ret                                     ; x
su_op_hold_holding:
    fstp    st0                             ;
    fld     dword [WRK+su_hold_wrk.holdval] ; x
    ret

%endif ; HOLD_ID > -1

;-------------------------------------------------------------------------------
;   CRUSH Tick
;-------------------------------------------------------------------------------
%if CRUSH_ID > -1

SECT_TEXT(sucrush)

EXPORT MANGLE_FUNC(su_op_crush,0)
    %ifdef INCLUDE_STEREO_CRUSH    
        call    su_effects_stereohelper
        %define INCLUDE_EFFECTS_STEREOHELPER
    %endif
    fdiv    dword [INP+su_crush_ports.resolution]
    frndint
    fmul    dword [INP+su_crush_ports.resolution]
    ret

%endif ; CRUSH_ID > -1

;-------------------------------------------------------------------------------
;   GAIN Tick
;-------------------------------------------------------------------------------
%if GAIN_ID > -1

SECT_TEXT(sugain)
    %ifdef INCLUDE_STEREO_GAIN
        EXPORT MANGLE_FUNC(su_op_gain,0)
            fld     dword [INP+su_gain_ports.gain] ; g l (r)
            jnc     su_op_gain_mono
            fmul    st2, st0                             ; g l r/g
        su_op_gain_mono:
            fmulp   st1, st0                             ; l/g (r/)
            ret
    %else
        EXPORT MANGLE_FUNC(su_op_gain,0)
            fmul    dword [INP+su_gain_ports.gain]
            ret
    %endif
%endif ; GAIN_ID > -1

;-------------------------------------------------------------------------------
;   INVGAIN Tick
;-------------------------------------------------------------------------------
%if INVGAIN_ID > -1

SECT_TEXT(suingain)
    %ifdef INCLUDE_STEREO_INVGAIN
        EXPORT MANGLE_FUNC(su_op_invgain,0)
            fld     dword [INP+su_invgain_ports.invgain] ; g l (r)
            jnc     su_op_invgain_mono
            fdiv    st2, st0                             ; g l r/g    
        su_op_invgain_mono:
            fdivp   st1, st0                             ; l/g (r/)
            ret
    %else
        EXPORT MANGLE_FUNC(su_op_invgain,0)
            fdiv    dword [INP+su_invgain_ports.invgain]
            ret
    %endif
%endif ; INVGAIN_ID > -1

;-------------------------------------------------------------------------------
;   su_op_filter: perform low/high/band-pass filtering on the signal
;-------------------------------------------------------------------------------
;   Input:      WRK     :   pointer to unit workspace
;               VAL     :   pointer to unit values as bytes
;               ecx     :   pointer to global workspace
;               st0     :   signal
;   Output:     st0     :   filtered signal
;   Dirty:      eax, edx
;-------------------------------------------------------------------------------
%if FILTER_ID > -1
SECT_TEXT(sufilter)

EXPORT MANGLE_FUNC(su_op_filter,0)
    lodsb ; load the flags to al
    %ifdef INCLUDE_STEREO_FILTER
        call    su_effects_stereohelper
        %define INCLUDE_EFFECTS_STEREOHELPER
    %endif
    fld     dword [INP+su_filter_ports.res] ; r x
    fld     dword [INP+su_filter_ports.freq]; f r x
    fmul    st0, st0                        ; f2 x (square the input so we never get negative and also have a smoother behaviour in the lower frequencies)
    fst     dword [_SP-4]                   ; f2 r x
    fmul    dword [WRK+su_filter_wrk.band]  ; f2*b r x
    fadd    dword [WRK+su_filter_wrk.low]   ; f2*b+l r x
    fst     dword [WRK+su_filter_wrk.low]   ; l'=f2*b+l r x
    fsubp   st2, st0                        ; r x-l'
    fmul    dword [WRK+su_filter_wrk.band]  ; r*b x-l'
    fsubp   st1, st0                        ; x-l'-r*b
    fst     dword [WRK+su_filter_wrk.high]  ; h'=x-l'-r*b
    fmul    dword [_SP-4]                   ; f2*h'
    fadd    dword [WRK+su_filter_wrk.band]  ; f2*h'+b
    fstp    dword [WRK+su_filter_wrk.band]  ; b'=f2*h'+b
    fldz                                    ; 0
%ifdef INCLUDE_LOWPASS
    test    al, byte LOWPASS
    jz      short su_op_filter_skiplowpass
    fadd    dword [WRK+su_filter_wrk.low]
su_op_filter_skiplowpass:
%endif
%ifdef INCLUDE_BANDPASS
    test    al, byte BANDPASS
    jz      short su_op_filter_skipbandpass
    fadd    dword [WRK+su_filter_wrk.band]
su_op_filter_skipbandpass:
%endif
%ifdef INCLUDE_HIGHPASS
    test    al, byte HIGHPASS
    jz      short su_op_filter_skiphighpass
    fadd    dword [WRK+su_filter_wrk.high]
su_op_filter_skiphighpass:
%endif
%ifdef INCLUDE_NEGBANDPASS
    test    al, byte NEGBANDPASS
    jz      short su_op_filter_skipnegbandpass
    fsub    dword [WRK+su_filter_wrk.band]
su_op_filter_skipnegbandpass:
%endif
%ifdef INCLUDE_NEGHIGHPASS
    test    al, byte NEGHIGHPASS
    jz      short su_op_filter_skipneghighpass
    fsub    dword [WRK+su_filter_wrk.high]
su_op_filter_skipneghighpass:
%endif
    ret
%endif ; SU_INCLUDE_FILTER

;-------------------------------------------------------------------------------
;   su_clip function
;-------------------------------------------------------------------------------
;   Input:      st0     :   x
;   Output:     st0     :   min(max(x,-1),1)
;-------------------------------------------------------------------------------
SECT_TEXT(suclip)

%if CLIP_ID > -1
    EXPORT MANGLE_FUNC(su_op_clip,0)
    %ifdef INCLUDE_STEREO_CLIP
        call    su_effects_stereohelper
        %define INCLUDE_EFFECTS_STEREOHELPER
    %endif
    %define SU_INCLUDE_CLIP
    ; flow into su_doclip
%endif ; CLIP_ID > -1

%ifdef SU_INCLUDE_CLIP
su_clip:
    fld1                                    ; 1 x a
    fucomi  st1                             ; if (1 <= x)
    jbe     short su_clip_do                ;   goto Clip_Do
    fchs                                    ; -1 x a
    fucomi  st1                             ; if (-1 < x)
    fcmovb  st0, st1                        ;   x x a
su_clip_do:
    fstp    st1                             ; x' a, where x' = clamp(x)
    ret

%endif ; SU_INCLUDE_CLIP

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
%if PAN_ID > -1

SECT_TEXT(supan)

%ifdef INCLUDE_STEREO_PAN

EXPORT MANGLE_FUNC(su_op_pan,0)
    jc      su_op_pan_do    ; this time, if this is mono op...
    fld     st0             ;   ...we duplicate the mono into stereo first
su_op_pan_do:
    fld     dword [INP+su_pan_ports.panning]    ; p l r
    fld1                                        ; 1 p l r
    fsub    st1                                 ; 1-p p l r
    fmulp   st2                                 ; p (1-p)*l r
    fmulp   st2                                 ; (1-p)*l p*r
    ret

%else ; ifndef INCLUDE_STEREO_PAN

EXPORT MANGLE_FUNC(su_op_pan,0)
    fld     dword [INP+su_pan_ports.panning]    ; p s
    fmul    st1                                 ; p*s s
    fsub    st1, st0                            ; p*s s-p*s
                                                ; Equal to
                                                ; s*p s*(1-p)
    fxch                                        ; s*(1-p) s*p SHOULD PROBABLY DELETE, WHY BOTHER
    ret

%endif ; INCLUDE_STEREO_PAN

%endif ; SU_USE_PAN

;-------------------------------------------------------------------------------
;   su_effects_stereohelper: moves the workspace to next, does the filtering for
;   right channel (pulling the calling address from stack), rewinds the
;   workspace and returns
;-------------------------------------------------------------------------------
%ifdef INCLUDE_EFFECTS_STEREOHELPER

su_effects_stereohelper:
    jnc     su_effects_stereohelper_mono ; carry is still the stereo bit
    add     WRK, 16
    fxch                  ; r l
    call    [_SP]         ; call whoever called me...
    fxch                  ; l r
    sub     WRK, 16       ; move WRK back to where it was
su_effects_stereohelper_mono:
    ret                   ; return to process l/mono sound

%endif

;-------------------------------------------------------------------------------
;   Delay Tick
;-------------------------------------------------------------------------------
;   Pseudocode:
;   q = dr*x
;   for (i = 0;i < count;i++)
;     s = b[(t-delaytime[i+offset])&65535]
;     q += s
;     o[i] = o[i]*da+s*(1-da)
;     b[t] = f*o[i] +p^2*x
;  Perform dc-filtering q and output
;-------------------------------------------------------------------------------
%if DELAY_ID > -1

SECT_TEXT(sudelay)

EXPORT MANGLE_FUNC(su_op_delay,0)
    lodsb             ; eax = delay index
    mov     edi, eax
    lodsb             ; eax = delay count
%ifdef INCLUDE_STEREO_DELAY
    jnc     su_op_delay_mono
    fxch
    call    su_op_delay_mono ; do right delay
    fxch
    add     edi, eax ; the second delay is done with the delay time index added by count
su_op_delay_mono:
%endif
    push_registers  _AX, _CX, _BX, WRK, _SI, _DI
    mov     ebx, edi; ugly register juggling, refactor
%ifdef DELAY_NOTE_SYNC
    test    ebx, ebx ; note s
    jne     su_op_delay_skipnotesync
    fld1
    fild    dword [_CX+su_unit.size-su_voice.workspace+su_voice.note]
    apply fmul dword, c_i12
    call    MANGLE_FUNC(su_power,0)
    apply fmul dword, c_freq_normalize ; // normalize
    fdivp   st1, st0                ; // invert to get numer of samples
    apply fistp word, MANGLE_DATA(su_delay_times)  ; store current comb size
su_op_delay_skipnotesync:
%endif
kmDLL_func_process:
    mov     ecx, eax                            ;// ecx is the number of parallel delays
    apply {mov WRK, PTRWORD},MANGLE_DATA(su_delay_buffer_ofs) ;// ebp is current delay
    fld     st0                                 ; x x
    fmul    dword [INP+su_delay_ports.dry]      ; dr*x x
    fxch                                        ; x dr*x
    fmul    dword [INP+su_delay_ports.pregain]  ; p*x dr*x
    fmul    dword [INP+su_delay_ports.pregain]  ; p^2*x dr*x

kmDLL_func_loop:
        mov     edi, dword [WRK + su_delayline_wrk.time]
        inc     edi
        and     edi, MAX_DELAY-1
        mov     dword [WRK + su_delayline_wrk.time],edi
        apply {movzx esi, word},MANGLE_DATA(su_delay_times),_BX*2,{} ; esi = comb size from the delay times table
        mov     eax, edi
        sub     eax, esi
        and     eax, MAX_DELAY-1
        apply fld dword, su_delayline_wrk.buffer, WRK, _AX*4,{} ; s p^2*x dr*x, where s is the sample from delay buffer
        ;// add comb output to current output
        fadd    st2, st0                            ; s p^2*x dr*x+s
        fld1                                        ; 1 s p^2*x dr*x+s
        fsub    dword [INP+su_delay_ports.damp]     ; 1-da s p^2*x dr*x+s
        fmulp   st1, st0                            ; s*(1-da) p^2*x dr*x+s
        fld     dword [INP+su_delay_ports.damp]     ; da s*(1-da) p^2*x dr*x+s
        fmul    dword [WRK+su_delayline_wrk.filtstate]      ; o*da s*(1-da) p^2*x dr*x+s, where o is stored
        faddp   st1, st0                            ; o*da+s*(1-da) p^2*x dr*x+s
        fst     dword [WRK+su_delayline_wrk.filtstate]      ; o'=o*da+s*(1-da), o' p^2*x dr*x+s
        fmul    dword [INP+su_delay_ports.feedback] ; f*o' p^2*x dr*x+s
        fadd    st0, st1                            ; f*o'+p^2*x p^2*x dr*x+s
        fstp    dword [WRK+_DI*4+su_delayline_wrk.buffer]; save f*o'+p^2*x to delay buffer
        inc     ebx                                 ;// go to next delay lenkmh index
        add     WRK, su_delayline_wrk.size               ;// go to next delay
        apply mov PTRWORD, MANGLE_DATA(su_delay_buffer_ofs),{, WRK} ;// store next delay offset
        loopne  kmDLL_func_loop
    fstp    st0                                 ; dr*x+s1+s2+s3+...
    ; DC-filtering
    sub     WRK, su_delayline_wrk.size ; the reason to use the last su_delayline_wrk instead of su_delay_wrk is that su_delay_wrk is wiped by retriggering
    fld     dword [WRK+su_delayline_wrk.dcout]  ; o s
    apply fmul dword, c_dc_const                      ; c*o s
    fsub    dword [WRK+su_delayline_wrk.dcin]   ; c*o-i s
    fxch                                    ; s c*o-i
    fst     dword [WRK+su_delayline_wrk.dcin]   ; i'=s, s c*o-i
    faddp   st1                             ; s+c*o-i
    apply fadd dword, c_0_5                       ;// add and sub small offset to prevent denormalization
    apply fsub dword, c_0_5
    fst     dword [WRK+su_delayline_wrk.dcout]  ; o'=s+c*o-i
    pop_registers  _AX, _CX, _BX, WRK, _SI, _DI
    ret

;-------------------------------------------------------------------------------
;    Delay data
;-------------------------------------------------------------------------------
SECT_BSS(sudelbuf)

EXPORT MANGLE_DATA(su_delay_buffer_ofs)
                        RESPTR  1

EXPORT MANGLE_DATA(su_delay_buffer)
                        resb   NUM_DELAY_LINES*su_delayline_wrk.size

SECT_DATA(suconst)

%ifndef C_DC_CONST
    c_dc_const              dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
    %define C_DC_CONST
%endif

%ifndef C_FREQ_NORMALIZE
    c_freq_normalize        dd      0.000092696138  ; // 220.0/(2^(69/12)) / 44100.0
    %define C_FREQ_NORMALIZE
%endif

%endif ; DELAY_ID > -1

;-------------------------------------------------------------------------------
;   Compressor Tick
;-------------------------------------------------------------------------------
%if COMPRES_ID > -1

SECT_TEXT(sucompr)

EXPORT MANGLE_FUNC(su_op_compressor,0)
    fld     st0                                 ; x x
    fmul    st0, st0                            ; x^2 x
%ifdef INCLUDE_STEREO_COMPRES
    jnc     su_op_compressor_mono
    fld     st2                                 ; r l^2 l r
    fmul    st0, st0                            ; r^2 l^2 l r
    faddp   st1, st0                            ; r^2+l^2 l r
    call    su_op_compressor_mono               ; So, for stereo, we square both left & right and add them up
    fld     st0                                 ; and return the computed gain two times, ready for MULP STEREO
    ret
su_op_compressor_mono:
%endif
    fld     dword [WRK+su_compres_wrk.level]    ; l x^2 x
    mov     al,1                                ; high bits are zero so this is ok. eax = 1 => release
    fucomi  st0, st1                            ; if l > x^2 // we're releasing
    jnb     su_op_compressor_releasing
    xor     eax, eax                            ; eax = 0 => attacking
su_op_compressor_releasing:
    fsub    st1, st0                            ; l x^2-l x
    call    su_env_map                          ; c l x^2-l x, c is either attack or release parameter mapped in a nonlinear way
    fmulp   st2, st0                            ; l c*(x^2-l) x
    faddp   st1, st0                            ; l+c*(x^2-l) x
    fst     dword [WRK+su_compres_wrk.level] ; l'=l+c*(x^2-l), l' x
    fld     dword [INP+su_compres_ports.threshold] ; t l' x
    fmul    st0, st0                            ; t*t
    fucomi  st0, st1                            ; if threshold < l'
    jb      su_op_compressor_compress           ;    then we actually do compression
    fstp    st0                                 ; l' x
    fstp    st0                                 ; x
    fld1                                        ; 1 x
    ret                                         ; return unity gain when we are below threshold
su_op_compressor_compress:                      ; l' x
    fdivrp  st1, st0                            ; t*t/l' x
    fld     dword [INP+su_compres_ports.ratio]  ; r t*t/l' x
    apply fmul dword, c_0_5                           ; p=r/2 t*t/l' x
    fxch                                        ; t*t/l' p x
    fyl2x                                       ; p*log2(t*t/l') x
    jmp     MANGLE_FUNC(su_power,0)             ; 2^(p*log2(t*t/l')) x
    ; tail call                                 ; Equal to:
                                                ; (t*t/l')^p x
                                                ; if ratio is at minimum => p=0 => 1 x
                                                ; if ratio is at maximum => p=0.5 => t/x => t/x*x=t

%endif ; COMPRES_ID > -1