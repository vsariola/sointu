%ifdef INCLUDE_GMDLS

%define SAMPLE_TABLE_SIZE 3440660 ; size of gmdls

extern _OpenFile@12 ; requires windows
extern _ReadFile@20 ; requires windows

SECT_TEXT(sugmdls)

EXPORT MANGLE_FUNC(su_load_gmdls,0)
    mov     edx, MANGLE_DATA(su_sample_table)
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
    push    SAMPLE_TABLE_SIZE       ; number of bytes to read
    push    edx                     ; here we actually pass the sample table to readfile
    push    eax                     ; handle to file
    call    _ReadFile@20            ; Readfile(handle,&su_sample_table,SAMPLE_TABLE_SIZE,&bytes_read,NULL)
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