{{- if .Opcode "speed" -}}
;-------------------------------------------------------------------------------
;   SPEED opcode: modulate the speed (bpm) of the song based on ST0
;-------------------------------------------------------------------------------
;   Mono: adds or subtracts the ticks, a value of 0.5 is neutral & will7
;   result in no speed change.
;   There is no STEREO version.
;-------------------------------------------------------------------------------
{{.Func "su_op_speed" "Opcode"}}
{{- .Float 2.206896551724138 | .Prepare | indent 4}}
    fmul    dword [{{.Float 2.206896551724138 | .Use}}]         ; (2*s-1)*64/24, let's call this p from now on
    {{.Call "su_power"}}
    fld1                                 ; 1 2^p
    fsubp   st1, st0                     ; 2^p-1, the player is advancing 1 tick by its own
    fadd    dword [{{.WRK}}] ; t+2^p-1, t is the remainder from previous rounds as ticks have to be rounded to 1
    push    {{.AX}}
    fist    dword [{{.SP}}]                  ; Main stack: k=int(t+2^p-1)
    fisub   dword [{{.SP}}]                  ; t+2^p-1-k, the remainder
    pop     {{.AX}}
    add     dword [{{.Stack "Sample"}}], eax          ; add the whole ticks to row tick count
    fstp    dword [{{.WRK}}] ; save the remainder for future
    ret
{{end}}
