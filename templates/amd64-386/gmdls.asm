{{- if .SupportsParamValue "oscillator" "type" .Sample}}

{{- if eq .OS "windows"}}
{{.ExportFunc "su_load_gmdls"}}
{{- if .Amd64}}
    extern OpenFile ; requires windows
    extern ReadFile ; requires windows
    ;        Win64 ABI: RCX, RDX, R8, and R9
    sub     rsp, 40         ; Win64 ABI requires "shadow space" + space for one parameter.
    mov     rdx, qword su_sample_table
    mov     rcx, qword su_gmdls_path1
    su_gmdls_pathloop:
        xor     r8,r8 ; OF_READ
        push    rdx                 ; &ofstruct, blatantly reuse the sample table
        push    rcx
        call    OpenFile            ; eax = OpenFile(path,&ofstruct,OF_READ)
        pop     rcx
        add     rcx, su_gmdls_path2 - su_gmdls_path1 ; if we ever get to third, then crash
        pop     rdx
        cmp     eax, -1             ; ecx == INVALID?
        je      su_gmdls_pathloop
    movsxd  rcx, eax
    mov     qword [rsp+32], 0
    mov     r9, rdx
    mov     r8d, 3440660   ; number of bytes to read
    call    ReadFile                ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
    add     rsp, 40         ; shadow space, as required by Win64 ABI
    ret
{{else}}
    mov     edx, su_sample_table
    mov     ecx, su_gmdls_path1
    su_gmdls_pathloop:
        push    0                   ; OF_READ
        push    edx                 ; &ofstruct, blatantly reuse the sample table
        push    ecx                 ; path
        call    _OpenFile@12        ; eax = OpenFile(path,&ofstruct,OF_READ)
        add     ecx, su_gmdls_path2 - su_gmdls_path1 ; if we ever get to third, then crash
        cmp     eax, -1             ; eax == INVALID?
        je      su_gmdls_pathloop
    push    0                       ; NULL
    push    edx                     ; &bytes_read, reusing sample table again; it does not matter that the first four bytes are trashed
    push    3440660                 ; number of bytes to read
    push    edx                     ; here we actually pass the sample table to readfile
    push    eax                     ; handle to file
    call    _ReadFile@20            ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
    ret
extern _OpenFile@12 ; requires windows
extern _ReadFile@20 ; requires windows
{{end}}

{{.Data "su_gmdls_path1"}}
    db 'drivers/gm.dls',0
su_gmdls_path2:
    db 'drivers/etc/gm.dls',0
{{end}}

{{.SectBss "susamtable"}}
su_sample_table:
    resb    3440660    ; size of gmdls.
{{end}}
