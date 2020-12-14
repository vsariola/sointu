{{- if not .Output16Bit }}
    {{- if not .Clip }}
            mov     {{.DI}}, [{{.Stack "OutputBufPtr"}}] ; edi containts ptr
            mov     {{.SI}}, {{.PTRWORD}} su_synth_obj + su_synthworkspace.left
            movsd   ; copy left channel to output buffer
            movsd   ; copy right channel to output buffer
            mov     [{{.Stack "OutputBufPtr"}}], {{.DI}} ; save back the updated ptr
            lea     {{.DI}}, [{{.SI}}-8]
            xor     eax, eax
            stosd   ; clear left channel so the VM is ready to write them again
            stosd   ; clear right channel so the VM is ready to write them again
    {{ else }}
            mov     {{.SI}}, qword [{{.Stack "OutputBufPtr"}}] ; esi points to the output buffer
            xor     ecx,ecx
            xor     eax,eax
            %%loop: ; loop over two channels, left & right
             do fld     dword [,su_synth_obj+su_synthworkspace.left,_CX*4,]
                {{.Call "su_clip"}}
                fstp    dword [_SI]
             do mov     dword [,su_synth_obj+su_synthworkspace.left,_CX*4,{],eax} ; clear the sample so the VM is ready to write it
                add     _SI,4
                cmp     ecx,2
                jl      %%loop
            mov     dword [_SP+su_stack.bufferptr - su_stack.output_sound], _SI ; save esi back to stack
    {{ end }}
{{- else}}
            mov     {{.SI}}, [{{.Stack "OutputBufPtr"}}] ; esi points to the output buffer
            mov     {{.DI}}, {{.PTRWORD}} su_synth_obj+su_synthworkspace.left
            mov     ecx, 2
            output_sound16bit_loop: ; loop over two channels, left & right
                    fld     dword [{{.DI}}]
                    {{.Call "su_clip"}}
            {{- .Float 32767.0 | .Prepare | indent 16}}
                    fmul    dword [{{.Float 32767.0 | .Use}}]
                    push    {{.AX}}
                    fistp   dword [{{.SP}}]
                    pop     {{.AX}}
                    mov     word [{{.SI}}],ax   ; // store integer converted right sample
                    xor     eax,eax
                    stosd
                    add     {{.SI}},2
                    loop    output_sound16bit_loop
            mov     [{{.Stack "OutputBufPtr"}}], {{.SI}} ; save esi back to stack
{{- end }}