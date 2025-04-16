{{- if .HasOp "hold"}}
;-------------------------------------------------------------------------------
;   HOLD opcode: sample and hold the signal, reducing sample rate
;-------------------------------------------------------------------------------
;   Mono version:   holds the signal at a rate defined by the freq parameter
;   Stereo version: holds both channels
;-------------------------------------------------------------------------------
{{.Func "su_op_hold" "Opcode"}}
{{- if .Stereo "hold"}}
    {{.Call "su_effects_stereohelper"}}
{{- end}}
    fld     dword [{{.Input "hold" "holdfreq"}}]    ; f x
    fmul    st0, st0                        ; f^2 x
    fchs                                    ; -f^2 x
    fadd    dword [{{.WRK}}]              ; p-f^2 x
    fst     dword [{{.WRK}}]              ; p <- p-f^2
    fldz                                    ; 0 p x
    fucomip st1                             ; p x
    fstp    dword [{{.SP}}-4]                   ; t=p, x
    jc      short su_op_hold_holding        ; if (0 < p) goto holding
    fld1                                    ; 1 x
    fadd    dword [{{.SP}}-4]                   ; 1+t x
    fstp    dword [{{.WRK}}]   ; x
    fst     dword [{{.WRK}}+4] ; save holded value
    ret                                     ; x
su_op_hold_holding:
    fstp    st0                             ;
    fld     dword [{{.WRK}}+4] ; x
    ret
{{end}}


{{- if .HasOp "crush"}}
;-------------------------------------------------------------------------------
;   CRUSH opcode: quantize the signal to finite number of levels
;-------------------------------------------------------------------------------
;   Mono:   x   ->  e*int(x/e)              where e=2**(-24*resolution)
;   Stereo: l r ->  e*int(l/e) e*int(r/e)
;-------------------------------------------------------------------------------
{{.Func "su_op_crush" "Opcode"}}
{{- if .Stereo "crush"}}
    {{.Call "su_effects_stereohelper"}}
{{- end}}
    xor     eax, eax
    {{.Call "su_nonlinear_map"}}
    fxch    st0, st1
    fdiv    st0, st1
    frndint
    fmulp   st1, st0
    ret
{{end}}


{{- if .HasOp "gain"}}
;-------------------------------------------------------------------------------
;   GAIN opcode: apply gain on the signal
;-------------------------------------------------------------------------------
;   Mono:   x   ->  x*g
;   Stereo: l r ->  l*g r*g
;-------------------------------------------------------------------------------
{{.Func "su_op_gain" "Opcode"}}
{{- if .Stereo "gain"}}
    fld     dword [{{.Input "gain" "gain"}}] ; g l (r)
{{- if .Mono "invgain"}}
    jnc     su_op_gain_mono
{{- end}}
    fmul    st2, st0                             ; g l r/g
su_op_gain_mono:
    fmulp   st1, st0                             ; l/g (r/)
    ret
{{- else}}
    fmul    dword [{{.Input "gain" "gain"}}]
    ret
{{- end}}
{{end}}


{{- if .HasOp "invgain"}}
;-------------------------------------------------------------------------------
;   INVGAIN opcode: apply inverse gain on the signal
;-------------------------------------------------------------------------------
;   Mono:   x   ->  x/g
;   Stereo: l r ->  l/g r/g
;-------------------------------------------------------------------------------
{{.Func "su_op_invgain" "Opcode"}}
{{- if .Stereo "invgain"}}
    fld     dword [{{.Input "invgain" "invgain"}}] ; g l (r)
{{- if .Mono "invgain"}}
    jnc     su_op_invgain_mono
{{- end}}
    fdiv    st2, st0                             ; g l r/g
su_op_invgain_mono:
    fdivp   st1, st0                             ; l/g (r/)
    ret
{{- else}}
    fdiv    dword [{{.Input "invgain" "invgain"}}]
    ret
{{- end}}
{{end}}

{{- if .HasOp "dbgain"}}
;-------------------------------------------------------------------------------
;   DBGAIN opcode: apply gain on the signal, with gain given in decibels
;-------------------------------------------------------------------------------
;   Mono:   x   ->  x*g, where g = 2**((2*d-1)*6.643856189774724) i.e. -40dB to 40dB, d=[0..1]
;   Stereo: l r ->  l*g r*g
;-------------------------------------------------------------------------------
{{.Func "su_op_dbgain" "Opcode"}}
{{- if .Stereo "dbgain"}}
    fld     dword [{{.Input "dbgain" "decibels"}}] ; d l r
{{- .Prepare (.Float 0.5)}}
    fsub    dword [{{.Use (.Float 0.5)}}]          ; d-.5
    fadd    st0, st0                               ; 2*d-1
{{- .Prepare (.Float 6.643856189774724)}}
    fmul    dword [{{.Use (.Float 6.643856189774724)}}] ; (2*d-1)*6.643856189774724
    {{.Call "su_power"}}
{{- if .Mono "dbgain"}}
    jnc     su_op_dbgain_mono
{{- end}}
    fmul    st2, st0                             ; g l r/g
su_op_dbgain_mono:
    fmulp   st1, st0                             ; l/g (r/)
    ret
{{- else}}
    fld     dword [{{.Input "dbgain" "decibels"}}] ; d l
{{- .Prepare (.Float 0.5)}}
    fsub    dword [{{.Use (.Float 0.5)}}]          ; d-.5
    fadd    st0, st0                               ; 2*d-1
{{- .Prepare (.Float 6.643856189774724)}}
    fmul    dword [{{.Use (.Float 6.643856189774724)}}] ; (2*d-1)*6.643856189774724
    {{.Call "su_power"}}
    fmulp   st1, st0
    ret
{{- end}}
{{end}}

{{- if .HasOp "filter"}}
;-------------------------------------------------------------------------------
;   FILTER opcode: perform low/high/band-pass/notch etc. filtering on the signal
;-------------------------------------------------------------------------------
;   Mono:   x   ->  filtered(x)
;   Stereo: l r ->  filtered(l) filtered(r)
;-------------------------------------------------------------------------------
{{.Func "su_op_filter" "Opcode"}}
    lodsb ; load the flags to al
{{- if .Stereo "filter"}}
    {{.Call "su_effects_stereohelper"}}
{{- end}}
    fld     dword [{{.Input "filter" "resonance"}}] ; r x
    fld     dword [{{.Input "filter" "frequency"}}]; f r x
    fmul    st0, st0                        ; f2 x (square the input so we never get negative and also have a smoother behaviour in the lower frequencies)
    fst     dword [{{.WRK}}+12]                   ; f2 r x
    fmul    dword [{{.WRK}}+8]  ; f2*b r x
    fadd    dword [{{.WRK}}]   ; f2*b+l r x
    fst     dword [{{.WRK}}]   ; l'=f2*b+l r x
    fsubp   st2, st0                        ; r x-l'
    fmul    dword [{{.WRK}}+8]  ; r*b x-l'
    fsubp   st1, st0                        ; x-l'-r*b
    {{- .Float 0.5 | .Prepare | indent 4}}
    fadd    dword [{{.Float 0.5 | .Use}}]           ; add and sub small offset to prevent denormalization
    fsub    dword [{{.Float 0.5 | .Use}}]           ; See for example: https://stackoverflow.com/questions/36781881/why-denormalized-floats-are-so-much-slower-than-other-floats-from-hardware-arch
    fst     dword [{{.WRK}}+4]  ; h'=x-l'-r*b
    fmul    dword [{{.WRK}}+12]                   ; f2*h'
    fadd    dword [{{.WRK}}+8]  ; f2*h'+b
    fstp    dword [{{.WRK}}+8]  ; b'=f2*h'+b
    fldz                                    ; 0
{{- if .SupportsParamValue "filter" "lowpass" 1}}
    test    al, byte 0x40
    jz      short su_op_filter_skiplowpass
    fadd    dword [{{.WRK}}]
su_op_filter_skiplowpass:
{{- end}}
{{- if .SupportsParamValue "filter" "bandpass" 1}}
    test    al, byte 0x20
    jz      short su_op_filter_skipbandpass
    fadd    dword [{{.WRK}}+8]
su_op_filter_skipbandpass:
{{- end}}
{{- if .SupportsParamValue "filter" "highpass" 1}}
    test    al, byte 0x10
    jz      short su_op_filter_skiphighpass
    fadd    dword [{{.WRK}}+4]
su_op_filter_skiphighpass:
{{- end}}
{{- if .SupportsParamValue "filter" "negbandpass" 1}}
    test    al, byte 0x08
    jz      short su_op_filter_skipnegbandpass
    fsub    dword [{{.WRK}}+8]
su_op_filter_skipnegbandpass:
{{- end}}
{{- if .SupportsParamValue "filter" "neghighpass" 1}}
    test    al, byte 0x04
    jz      short su_op_filter_skipneghighpass
    fsub    dword [{{.WRK}}+4]
su_op_filter_skipneghighpass:
{{- end}}
    ret
{{end}}


{{- if .HasOp "clip"}}
;-------------------------------------------------------------------------------
;   CLIP opcode: clips the signal into [-1,1] range
;-------------------------------------------------------------------------------
;   Mono:   x   ->  min(max(x,-1),1)
;   Stereo: l r ->  min(max(l,-1),1) min(max(r,-1),1)
;-------------------------------------------------------------------------------
{{.Func "su_op_clip" "Opcode"}}
{{- if .Stereo "clip"}}
    {{.Call "su_effects_stereohelper"}}
{{- end}}
    {{.TailCall "su_clip"}}
{{end}}


{{- if .HasOp "pan" -}}
;-------------------------------------------------------------------------------
;   PAN opcode: pan the signal
;-------------------------------------------------------------------------------
;   Mono:   s   ->  s*(1-p) s*p
;   Stereo: l r ->  l*(1-p) r*p
;
;   where p is the panning in [0,1] range
;-------------------------------------------------------------------------------
{{.Func "su_op_pan" "Opcode"}}
{{- if .Stereo "pan"}}
    jc      su_op_pan_do    ; this time, if this is mono op...
    fld     st0             ;   ...we duplicate the mono into stereo first
su_op_pan_do:
    fld     dword [{{.Input "pan" "panning"}}]    ; p l r
    fld1                                        ; 1 p l r
    fsub    st1                                 ; 1-p p l r
    fmulp   st2                                 ; p (1-p)*l r
    fmulp   st2                                 ; (1-p)*l p*r
    ret
{{- else}}
    fld     dword [{{.Input "pan" "panning"}}]    ; p s
    fmul    st1                                 ; p*s s
    fsub    st1, st0                            ; p*s s-p*s
                                                ; Equal to
                                                ; s*p s*(1-p)
    fxch                                        ; s*(1-p) s*p SHOULD PROBABLY DELETE, WHY BOTHER
    ret
{{- end}}
{{end}}


{{- if .HasOp "delay"}}
;-------------------------------------------------------------------------------
;   DELAY opcode: adds delay effect to the signal
;-------------------------------------------------------------------------------
;   Mono:   perform delay on ST0, using delaycount delaylines starting
;           at delayindex from the delaytable
;   Stereo: perform delay on ST1, using delaycount delaylines starting
;           at delayindex + delaycount from the delaytable (so the right delays
;           can be different)
;-------------------------------------------------------------------------------
{{.Func "su_op_delay" "Opcode"}}
    lodsw                           ; al = delay index, ah = delay count
    {{- .PushRegs .VAL "DelayVal" .COM "DelayCom" | indent 4}}
    movzx   ebx, al
{{- if .Library}}
    mov     {{.SI}}, [{{.Stack "DelayTable"}}] ; when using runtime tables, delaytimes is pulled from the stack so can be a pointer to heap
    lea     {{.BX}}, [{{.SI}} + {{.BX}}*2]
{{- else}}
{{- .Prepare "su_delay_times" | indent 4}}
    lea     {{.BX}},[{{.Use "su_delay_times"}} + {{.BX}}*2]                  ; BX now points to the right position within delay time table
{{- end}}
    movzx   esi, word [{{.Stack "GlobalTick"}}]          ; notice that we load word, so we wrap at 65536
    mov     {{.CX}}, {{.PTRWORD}} [{{.Stack "DelayWorkSpace"}}]   ; {{.WRK}} is now the separate delay workspace, as they require a lot more space
{{- if .StereoAndMono "delay"}}
    jnc     su_op_delay_mono
{{- end}}
{{- if .Stereo "delay"}}
    push    {{.AX}}                 ; save _ah (delay count)
    fxch                        ; r l
    call    su_op_delay_do      ; D(r) l        process delay for the right channel
    pop     {{.AX}}                 ; restore the count for second run
    fxch                        ; l D(r)
su_op_delay_mono:               ; flow into mono delay
{{- end}}
    call    su_op_delay_do      ; when stereo delay is not enabled, we could inline this to save 5 bytes, but I expect stereo delay to be farely popular so maybe not worth the hassle
    mov     {{.PTRWORD}} [{{.Stack "DelayWorkSpace"}}],{{.CX}}   ; move delay workspace pointer back to stack.
    {{- .PopRegs .VAL .COM | indent 4}}
{{- if .SupportsModulation "delay" "delaytime"}}
    xor     eax, eax
    mov     dword [{{.Modulation "delay" "delaytime"}}], eax
{{- end}}
    ret

;-------------------------------------------------------------------------------
;   su_op_delay_do: executes the actual delay
;-------------------------------------------------------------------------------
;   Pseudocode:
;   q = dr*x
;   for (i = 0;i < count;i++)
;     s = b[(t-delaytime[i+offset])&65535]
;     q += s
;     o[i] = o[i]*da+s*(1-da)
;     b[t] = f*o[i] +p^2*x
;  Perform dc-filtering q and output q
;-------------------------------------------------------------------------------
{{.Func "su_op_delay_do"}}                         ; x y
    fld     st0
    fmul    dword [{{.Input "delay" "pregain"}}]  ; p*x y
    fmul    dword [{{.Input "delay" "pregain"}}]  ; p*p*x y
    fxch                                        ; y p*p*x
    fmul    dword [{{.Input "delay" "dry"}}]      ; dr*y p*p*x
su_op_delay_loop:
        {{- if or (.SupportsModulation "delay" "delaytime") (.SupportsParamValue "delay" "notetracking" 1)}} ; delaytime modulation or note syncing require computing the delay time in floats
        fild    word [{{.BX}}]         ; k dr*y p*p*x, where k = delay time
        {{- if .SupportsModulation "delay" "delaytime"}}
        fld     dword [{{.Modulation "delay" "delaytime"}}]
        {{- .Float 32767.0 | .Prepare | indent 8}}
        fmul    dword [{{.Float 32767.0 | .Use}}] ; scale it up, as the modulations would be too small otherwise
        faddp   st1, st0
        {{- end}}
        {{- if .SupportsParamValue "delay" "notetracking" 1}}
        test    ah, 1 ; note syncing is the least significant bit of ah, 0 = ON, 1 = OFF
        jne     su_op_delay_skipnotesync
        fild    dword [{{.INP}}-su_voice.inputs+su_voice.note]
        {{.Int 0x3DAAAAAA | .Prepare | indent 8}}
        fmul    dword [{{.Int 0x3DAAAAAA | .Use}}]
        {{.Call "su_power"}}
        fdivp   st1, st0                 ; use 10787 for delaytime to have neutral transpose
        su_op_delay_skipnotesync:
        {{- end}}
        fistp   dword [{{.SP}}-4]                       ; dr*y p*p*x, dword [{{.SP}}-4] = integer amount of delay (samples)
        mov     edi, esi                            ; edi = esi = current time
        sub     di, word [{{.SP}}-4]                    ; we perform the math in 16-bit to wrap around
        {{- else}}
        mov     edi, esi
        sub     di, word [{{.BX}}]                      ; we perform the math in 16-bit to wrap around
        {{- end}}
        fld     dword [{{.CX}}+su_delayline_wrk.buffer+{{.DI}}*4]; s dr*y p*p*x, where s is the sample from delay buffer
        fadd    st1, st0                                ; s dr*y+s p*p*x (add comb output to current output)
        fld1                                            ; 1 s dr*y+s p*p*x
        fsub    dword [{{.Input "delay" "damp"}}]         ; 1-da s dr*y+s p*p*x
        fmulp   st1, st0                                ; s*(1-da) dr*y+s p*p*x
        fld     dword [{{.Input "delay" "damp"}}]         ; da s*(1-da) dr*y+s p*p*x
        fmul    dword [{{.CX}}+su_delayline_wrk.filtstate]  ; o*da s*(1-da) dr*y+s p*p*x, where o is stored
        faddp   st1, st0                                ; o*da+s*(1-da) dr*y+s p*p*x
        {{- .Float 0.5 | .Prepare | indent 4}}
        fadd    dword [{{.Float 0.5 | .Use}}]           ; add and sub small offset to prevent denormalization. WARNING: this is highly important, as the damp filters might denormalize and give 100x CPU penalty
        fsub    dword [{{.Float 0.5 | .Use}}]           ; See for example: https://stackoverflow.com/questions/36781881/why-denormalized-floats-are-so-much-slower-than-other-floats-from-hardware-arch
        fst     dword [{{.CX}}+su_delayline_wrk.filtstate]  ; o'=o*da+s*(1-da), o' dr*y+s p*p*x
        fmul    dword [{{.Input "delay" "feedback"}}]     ; f*o' dr*y+s p*p*x
        fadd    st0, st2                                ; f*o'+p*p*x dr*y+s p*p*x
        fstp    dword [{{.CX}}+su_delayline_wrk.buffer+{{.SI}}*4]; save f*o'+p*p*x to delay buffer
        add     {{.BX}},2                                   ; move to next index
        add     {{.CX}}, su_delayline_wrk.size              ; go to next delay delay workspace
        sub     ah, 2
        jg      su_op_delay_loop                        ; if ah > 0, goto loop
    fstp    st1                                 ; dr*y+s1+s2+s3+...
    ; DC-filtering
    fld     dword [{{.CX}}+su_delayline_wrk.dcout]  ; o s
{{- .Float 0.99609375 | .Prepare | indent 4}}
    fmul    dword [{{.Float 0.99609375 | .Use}}]                ; c*o s
    fsub    dword [{{.CX}}+su_delayline_wrk.dcin]   ; c*o-i s
    fxch                                        ; s c*o-i
    fst     dword [{{.CX}}+su_delayline_wrk.dcin]   ; i'=s, s c*o-i
    faddp   st1                                 ; s+c*o-i
{{- .Float 0.5 | .Prepare | indent 4}}
    fadd    dword [{{.Float 0.5 | .Use}}]          ; add and sub small offset to prevent denormalization. WARNING: this is highly important, as low pass filters might denormalize and give 100x CPU penalty
    fsub    dword [{{.Float 0.5 | .Use}}]          ; See for example: https://stackoverflow.com/questions/36781881/why-denormalized-floats-are-so-much-slower-than-other-floats-from-hardware-arch
    fst     dword [{{.CX}}+su_delayline_wrk.dcout]  ; o'=s+c*o-i
    ret
{{end}}


{{- if .HasOp "compressor"}}
;-------------------------------------------------------------------------------
;   COMPRES opcode: push compressor gain to stack
;-------------------------------------------------------------------------------
;   Mono:   push g on stack, where g is a suitable gain for the signal
;           you can either MULP to compress the signal or SEND it to a GAIN
;           somewhere else for compressor side-chaining.
;   Stereo: push g g on stack, where g is calculated using l^2 + r^2
;-------------------------------------------------------------------------------
{{.Func "su_op_compressor" "Opcode"}}
    fld     st0                                 ; x x
    fmul    st0, st0                            ; x^2 x
{{- if .StereoAndMono "compressor"}}
    jnc     su_op_compressor_mono
{{- end}}
{{- if .Stereo "compressor"}}
    fld     st2                                 ; r x^2 l r
    fst     st3                                 ; y x^2 l r
    fmul    st0, st0                            ; y^2 x^2 l r
    faddp   st1, st0                            ; y^2+x^2 l r
{{- if .StereoAndMono "compressor"}}
    call    su_op_compressor_mono               ; So, for stereo, we square both left & right and add them up
    fld     st0                                 ; and return the computed gain two times, ready for MULP STEREO
    ret
su_op_compressor_mono:
{{- end}}
{{- end}}
    fld     dword [{{.WRK}}]    ; l x^2 x
    fucomi  st0, st1
    setnb   al                                  ; if (st0 >= st1) al = 1; else al = 0;
    fsubp   st1, st0                            ; x^2-l x
    {{.Call "su_nonlinear_map"}}                ; c x^2-l x, c is either attack or release parameter mapped in a nonlinear way
    fmulp   st1, st0                            ; c*(x^2-l) x
    fadd    dword [{{.WRK}}]    ; l+c*(x^2-l) x   // we could've kept level in the stack and save a few bytes, but su_env_map uses 3 stack (c + 2 temp), so the stack was getting quite big.
    ; TODO: make this denormalization optional, if the user wants to save some space
    {{- .Float 0.5 | .Prepare | indent 4}}
    fadd    dword [{{.Float 0.5 | .Use}}]           ; add and sub small offset to prevent denormalization. WARNING: this is highly important, as the damp filters might denormalize and give 100x CPU penalty
    fsub    dword [{{.Float 0.5 | .Use}}]           ; See for example: https://stackoverflow.com/questions/36781881/why-denormalized-floats-are-so-much-slower-than-other-floats-from-hardware-arch
    fst     dword [{{.WRK}}]    ; l'=l+c*(x^2-l), l' x
    fld     dword [{{.Input "compressor" "threshold"}}] ; t l' x
    fmul    st0, st0                            ; t*t l' x
    fxch                                        ; l' t*t x
    fucomi  st0, st1                            ; if l' < t*t
    fcmovb  st0, st1                            ;   l'=t*t
    fdivp   st1, st0                            ; t*t/l' x
    fld     dword [{{.Input "compressor" "ratio"}}]  ; r t*t/l' x
{{.Float 0.5 | .Prepare | indent 4}}
    fmul    dword [{{.Float 0.5 | .Use}}]       ; p=r/2 t*t/l' x
    fxch                                        ; t*t/l' p x
    fyl2x                                       ; p*log2(t*t/l') x
    {{.Call "su_power"}}                        ; 2^(p*log2(t*t/l')) x
                                                ; Equal to:
                                                ; (t*t/l')^p x
                                                ; if ratio is at minimum => p=0 => 1 x
                                                ; if ratio is at maximum => p=0.5 => t/x => t/x*x=t
    fdiv    dword [{{.Input "compressor" "invgain"}}]; this used to be pregain but that ran into problems with getting back up to 0 dB so postgain should be better at that
{{- if and (.Stereo "compressor") (not (.Mono "compressor"))}}
    fld     st0                                 ; and return the computed gain two times, ready for MULP STEREO
{{- end}}
    ret
{{- end}}
