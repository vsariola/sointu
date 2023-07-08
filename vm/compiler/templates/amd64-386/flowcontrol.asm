{{- if .HasOp "speed" -}}
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


{{- if or .RowSync (.HasOp "sync")}}
;-------------------------------------------------------------------------------
;   SYNC opcode: save the stack top to sync buffer
;-------------------------------------------------------------------------------
{{.Func "su_op_sync" "Opcode"}}
{{- if not .Library}}
    ; TODO: syncs are NOPs when compiling as library, should figure out a way to
    ; make them work when compiling to use the native track also
    mov     {{.AX}}, [{{.Stack "GlobalTick"}}]
    test    al, al
    jne     su_op_sync_skip
    xchg    {{.AX}}, [{{.Stack "SyncBufPtr"}}]
    fst     dword [{{.AX}}]
    add     {{.AX}}, 4
    xchg    {{.AX}}, [{{.Stack "SyncBufPtr"}}]
su_op_sync_skip:
{{- end}}
    ret
{{end}}
