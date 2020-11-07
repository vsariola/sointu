%ifdef INCLUDE_GMDLS

%define SAMPLE_TABLE_SIZE 3440660 ; size of gmdls

extern OpenFile ; requires windows
extern ReadFile ; requires windows

SECT_TEXT(sugmdls)

EXPORT MANGLE_FUNC(su_load_gmdls,0)
;        Win64 ABI: RCX, RDX, R8, and R9
    sub     rsp, 40         ; Win64 ABI requires "shadow space" + space for one parameter.
    mov     rdx, PTRWORD MANGLE_DATA(su_sample_table)
    mov     rcx, PTRWORD su_gmdls_path1
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
    mov     qword [rsp+32],0
    mov     r9, rdx
    mov     r8d, SAMPLE_TABLE_SIZE   ; number of bytes to read
    call    ReadFile                ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
    add     rsp, 40         ; shadow space, as required by Win64 ABI
    ret

SECT_DATA(sugmpath)

su_gmdls_path1:
    db 'drivers/gm.dls',0
su_gmdls_path2:
    db 'drivers/etc/gm.dls',0

SECT_BSS(susamtbl)

EXPORT MANGLE_DATA(su_sample_table)
    resb    SAMPLE_TABLE_SIZE    ; size of gmdls.

%endif