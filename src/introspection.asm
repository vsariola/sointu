; Various compile time definitions exported
SECT_DATA(introscn)

%ifdef SU_USE_16BIT_OUTPUT
    EXPORT MANGLE_DATA(su_use_16bit_output) dd 1
%else
    EXPORT MANGLE_DATA(su_use_16bit_output) dd 0
%endif

%ifdef MAX_SAMPLES
    EXPORT MANGLE_DATA(su_max_samples)  dd MAX_SAMPLES
%endif
