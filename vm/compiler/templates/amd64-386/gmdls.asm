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
    xor     r8,r8 ; OF_READ
    push    rdx                 ; &ofstruct, blatantly reuse the sample table
    push    rcx
    call    OpenFile            ; eax = OpenFile(path,&ofstruct,OF_READ)
    pop     rcx
    pop     rdx
    movsxd  rcx, eax
    mov     qword [rsp+32], 0
    mov     r9, rdx
    mov     r8d, 3440660   ; number of bytes to read
    call    ReadFile                ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
    add     rsp, 40         ; shadow space, as required by Win64 ABI
    ret
{{else}}
    mov     ebx, su_sample_table
    push    0                   ; OF_READ
    push    ebx                 ; &ofstruct, blatantly reuse the sample table
    push    su_gmdls_path1      ; path
    call    dword [__imp__OpenFile@12]; eax = OpenFile(path,&ofstruct,OF_READ) // should not touch ebx according to calling convention
    push    0                       ; NULL
    push    ebx                     ; &bytes_read, reusing sample table again; it does not matter that the first four bytes are trashed
    push    3440660                 ; number of bytes to read
    push    ebx                     ; here we actually pass the sample table to readfile
    push    eax                     ; handle to file
    call    dword [__imp__ReadFile@20] ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
    ret
extern __imp__OpenFile@12 ; requires windows
extern __imp__ReadFile@20
 ; requires windows
{{end}}

{{.Data "su_gmdls_path1"}}
    db 'drivers/gm.dls',0
{{end}}

{{.SectBss "susamtable"}}
su_sample_table:
    resb    3440660    ; size of gmdls.
{{end}}
