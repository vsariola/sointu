%ifdef INCLUDE_GMDLS

%define SAMPLE_TABLE_SIZE 3440660 ; size of gmdls

extern OpenFile ; requires windows
extern ReadFile ; requires windows

SECT_TEXT(sugmdls)
;        Win64 ABI: RCX, RDX, R8, and R9
su_gmdls_load:
    sub     rsp, 40         ; Win64 ABI requires "shadow space" + space for one parameter.
    mov     rdi, PTRWORD MANGLE_DATA(su_sample_table)
    mov     rsi, PTRWORD su_gmdls_path1
    su_gmdls_pathloop:
        xor     r8,r8 ; OF_READ
        mov     rdx, rdi             ; &ofstruct, blatantly reuse the sample table
        mov     rcx, rsi        ; path
        call    OpenFile            ; eax = OpenFile(path,&ofstruct,OF_READ)
        add     rsi, su_gmdls_path2 - su_gmdls_path1 ; if we ever get to third, then crash
        movsxd  rcx,eax
        cmp     rcx, -1             ; ecx == INVALID?
        je      su_gmdls_pathloop
    mov     qword [rsp+32],0
    mov     r9, rdi
    mov     r8d, SAMPLE_TABLE_SIZE   ; number of bytes to read
    mov     rdx, rdi
    call    ReadFile                ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
    add     rsp, 40         ; shadow space, as required by Win64 ABI
    ret

SECT_DATA(sugmpath)

su_gmdls_path1:
    db 'drivers/gm.dls',0
su_gmdls_path2:
    db 'drivers/etc/gm.dls',0

SECT_DATA(suconst)
    c_samplefreq_scaling    dd      84.28074964676522       ; o = 0.000092696138, n = 72, f = 44100*o*2**(n/12), scaling = 22050/f <- so note 72 plays at the "normal rate"

SECT_BSS(susamtbl)
    EXPORT MANGLE_DATA(su_sample_table)    resb    SAMPLE_TABLE_SIZE    ; size of gmdls.

%endif