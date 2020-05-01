%ifndef _4KLANG_INC
%define _4KLANG_INC

; Following defines have to be defined before this include:
; MAX_INSTRUMENTS       e.g.         %define MAX_INSTRUMENTS 10
; BPM                   e.g.         %define BPM 100
; MAX_PATTERNS          e.g.         %define MAX_PATTERNS 1
;
; Optionally:
; PATTERN_SIZE_SHIFT    e.g.         %define PATTERN_SIZE_SHIFT 4 <- this is the default

%macro EXPORT 1
    global %1
    %1
%endmacro		

%ifidn __OUTPUT_FORMAT__,win32
    ; on win32, function f with n parameters is mangled as "_f@n"
    %define MANGLE_FUNC(f,n) _ %+ f %+ @ %+ n
    %define WIN_OR_MAC
%endif 

%ifidn __OUTPUT_FORMAT__,elf32 
    ; on linux, function f with n parameters is mangled as "f"
    %define MANGLE_FUNC(f,n) f
%endif 

%ifidn __OUTPUT_FORMAT__,macho32
    ; on mac, function f with x parameters is mangled as "_f"
    %define MANGLE_FUNC(f,n) _f
    %define WIN_OR_MAC
%endif 

%ifdef WIN_OR_MAC
    ; Windows has crinkler so one may USE_SECTIONS to put everything in custom sections to aid crinkler.
    ; Maybe mac users need it too
    %ifdef USE_SECTIONS
        %define SECT_BSS(n) section . %+ n bss align=1
        %define SECT_DATA(n) section . %+ n data align=1
        %define SECT_TEXT(n) section . %+ n code align=1
    %else
        %define SECT_BSS(n) section .bss align=1
        %define SECT_DATA(n) section .data align=1
        %define SECT_TEXT(n) section .code align=1
    %endif
    ; On windows and mac, data label d is mangled as "_d"
    %define MANGLE_DATA(d) _ %+ d
%else
    ; Linux
    %ifdef USE_SECTIONS
        %define SECT_BSS(n) section .bss. %+ n nobits alloc noexec write align=1
        %define SECT_DATA(n) section .data. %+ n progbits alloc noexec write align=1
        %define SECT_TEXT(n) section .text. %+ n progbits alloc exec nowrite align=1
    %else
        %define SECT_BSS(n) section .bss. nobits alloc noexec write align=1
        %define SECT_DATA(n) section .data. progbits alloc noexec write align=1
        %define SECT_TEXT(n) section .text. progbits alloc exec nowrite align=1
    %endif
    ; On linux, data label d is mangled as "d"
    %define MANGLE_DATA(d) d
%endif

%ifdef GO4K_USE_ALL
    ; GO4K_USE_ALL is convenience way to enable almost all features of the synth, for vsti plugins and such which
    ; do not have any size constraints. However, GO4K_USE_ALL should only enable features that absolutely do not
    ; change the functioning of the synth in any way, just add features. Clipping, 16 bit output etc. should still
    ; be enabled only whent they are actually needed

    ; Things that are NOT defined by GO4K_USE_ALL
    ;%define GO4K_USE_16BIT_OUTPUT         ; // removing this will output to 32bit floating point buffer
    ;%define GO4K_USE_GROOVE_PATTERN       ; // removing this skips groove pattern code
    ;%define GO4K_USE_ENVELOPE_RECORDINGS  ; // removing this skips envelope recording code
    ;%define GO4K_USE_NOTE_RECORDINGS      ; // removing this skips note recording code
    ;%define GO4K_USE_UNDENORMALIZE        ; // removing this skips denormalization code in the units
    ;%define GO4K_CLIP_OUTPUT              ; // removing this skips clipping code for the final output

    %define GO4K_USE_DST                    ; // removing this will skip DST unit
    %define GO4K_USE_DLL                    ; // removing this will	skip DLL unit
    %define GO4K_USE_PAN                    ; // removing this will	skip PAN unit
    %define GO4K_USE_GLOBAL_DLL             ; // removing this will	skip global	dll	processing
    %define GO4K_USE_FSTG                   ; // removing this will	skip global	store unit
    %define GO4K_USE_FLD                    ; // removing this will	skip float load unit
    %define GO4K_USE_GLITCH                 ; // removing this will	skip GLITCH unit
    %define GO4K_USE_ENV_CHECK              ; // removing this skips checks	if processing is needed
    %define GO4K_USE_VCO_CHECK              ; // removing this skips checks	if processing is needed
    %define GO4K_USE_VCO_PHASE_OFFSET       ; // removing this will	skip initial phase offset code
    %define GO4K_USE_VCO_SHAPE				; // removing this skips waveshaping code
    %define GO4K_USE_VCO_GATE				; // removing this skips gate code
    %define GO4K_USE_VCO_MOD_FM				; // removing this skips frequency modulation code
    %define GO4K_USE_VCO_MOD_DM				; // removing this skips detune	modulation code
    %define GO4K_USE_VCO_STEREO				; // removing this skips stereo code
    %define GO4K_USE_VCF_CHECK				; // removing this skips checks	if processing is needed
    %define GO4K_USE_VCF_HIGH				; // removing this skips code for high output
    %define GO4K_USE_VCF_BAND				; // removing this skips code for band output
    %define GO4K_USE_VCF_PEAK				; // removing this skips code for peak output
    %define GO4K_USE_VCF_STEREO				; // removing this skips code for stereo filter output
    %define GO4K_USE_DST_CHECK				; // removing this skips checks	if processing is needed
    %define GO4K_USE_DST_SH					; // removing this skips sample	and	hold code
    %define GO4K_USE_DST_STEREO             ; // removing this skips stereo processing
    %define GO4K_USE_DLL_NOTE_SYNC          ; // removing this will	skip delay length adjusting	code (karplus strong)
    %define GO4K_USE_DLL_CHORUS             ; // removing this will	skip delay chorus/flanger code
    %define GO4K_USE_DLL_CHORUS_CLAMP       ; // removing this will skip chorus lfo phase clamping
    %define GO4K_USE_DLL_DAMP				; // removing this will	skip dll damping code
    %define GO4K_USE_DLL_DC_FILTER			; // removing this will	skip dll dc	offset removal code
    %define GO4K_USE_FSTG_CHECK				; // removing this skips checks	if processing is needed
    %define GO4K_USE_WAVESHAPER_CLIP		; // removing this will	skip clipping code    
%endif

%ifdef GO4K_USE_VCO_SHAPE
    %define INCLUDE_WAVESHAPER
%endif
%ifdef GO4K_USE_DST
    %define INCLUDE_WAVESHAPER
%endif

%ifdef GO4K_USE_ENVELOPE_RECORDINGS
    %define GO4K_USE_BUFFER_RECORDINGS
%endif
%ifdef GO4K_USE_NOTE_RECORDINGS
    %define GO4K_USE_BUFFER_RECORDINGS
%endif

; //----------------------------------------------------------------------------------------
; // synth defines
; //----------------------------------------------------------------------------------------

%define	MAX_DELAY			65536
%define MAX_UNITS			64
%define	MAX_UNIT_SLOTS		16
%define MAX_WORK_VARS       8

%ifndef SAMPLE_RATE
%define SAMPLE_RATE 44100
%endif 

%ifndef MAX_VOICES
%define	MAX_VOICES 1
%endif 

%ifndef HLD
%define	HLD	1
%endif 

%ifndef PATTERN_SIZE_SHIFT
%define PATTERN_SIZE_SHIFT 4
%endif

%define	PATTERN_SIZE (1	<< PATTERN_SIZE_SHIFT)
%define	MAX_TICKS (MAX_PATTERNS*PATTERN_SIZE)
%define	SAMPLES_PER_TICK (SAMPLE_RATE*4*60/(BPM*16))
%define	DEF_LFO_NORMALIZE 0.000038
%define	MAX_SAMPLES	(SAMPLES_PER_TICK*MAX_TICKS)

%define GO4K_BEGIN_CMDDEF(def_name)
%define GO4K_END_CMDDEF db 0
%define GO4K_BEGIN_PARAMDEF(def_name)
%define GO4K_END_PARAMDEF

; //----------------------------------------------------------------------------------------
; // ENV structs
; //----------------------------------------------------------------------------------------
GO4K_ENV_ID		equ		1
%macro	GO4K_ENV 5
    db	%1
    db	%2
    db	%3
    db	%4
    db	%5
%endmacro
%define	ATTAC(val)		val	
%define	DECAY(val)		val	
%define	SUSTAIN(val)	val	
%define	RELEASE(val)	val	
%define	GAIN(val)		val	
struc	go4kENV_val
;//	unit paramters
    .attac		resd	1
    .decay		resd	1
    .sustain	resd	1
    .release	resd	1
    .gain		resd	1
    .size
endstruc
struc	go4kENV_wrk
;//	work variables	
    .state		resd	1
    .level		resd	1
    .size
endstruc
%define	ENV_STATE_ATTAC		0
%define	ENV_STATE_DECAY		1
%define	ENV_STATE_SUSTAIN	2
%define	ENV_STATE_RELEASE	3
%define	ENV_STATE_OFF		4
; //----------------------------------------------------------------------------------------
; // VCO structs
; //----------------------------------------------------------------------------------------
GO4K_VCO_ID		equ		2

%macro	GO4K_VCO 8
    db	%1
    db	%2
%ifdef GO4K_USE_VCO_PHASE_OFFSET	
    db	%3
%endif
%ifdef GO4K_USE_VCO_GATE
    db	%4
%endif	
    db	%5
%ifdef GO4K_USE_VCO_SHAPE
    db	%6
%endif	
    db	%7
    db	%8
%endmacro
%define	TRANSPOSE(val)	val	
%define	DETUNE(val)		val	
%define	PHASE(val)		val
%define GATES(val)		val		
%define	COLOR(val)		val
%define	SHAPE(val)		val	
%define	FLAGS(val)		val	
%define	SINE		0x01
%define	TRISAW		0x02
%define	PULSE		0x04
%define	NOISE		0x08
%define	LFO			0x10
%define	GATE		0x20
%define	VCO_STEREO	0x40
struc	go4kVCO_val
;//	unit paramters
    .transpose	resd	1
    .detune		resd	1
%ifdef GO4K_USE_VCO_PHASE_OFFSET	
    .phaseofs	resd	1
%endif	
%ifdef GO4K_USE_VCO_GATE
    .gate		resd	1
%endif	
    .color		resd	1
%ifdef GO4K_USE_VCO_SHAPE	
    .shape		resd	1
%endif
    .gain		resd	1
    .flags		resd	1	
    .size
endstruc
struc	go4kVCO_wrk
;//	work variables
    .phase		resd	1
;// stero variables	
    .phase2		resd	1
    .gatestate  resd    1
    .detune_mod resd    1
    .freq_mod   resd    1
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // VCF structs
; //----------------------------------------------------------------------------------------
GO4K_VCF_ID		equ		3
%macro	GO4K_VCF 3
    db	%1
    db	%2
    db	%3
%endmacro
%define	LOWPASS		0x1
%define	HIGHPASS	0x2
%define	BANDPASS	0x4
%define	BANDSTOP	0x3
%define	ALLPASS		0x7
%define	PEAK		0x8
%define STEREO		0x10
%define	FREQUENCY(val)	val
%define	RESONANCE(val)	val
%define	VCFTYPE(val)	val
struc	go4kVCF_val
;//	unit paramters
    .freq		resd	1
    .res		resd	1
    .type		resd	1
    .size
endstruc
struc	go4kVCF_wrk
;//	work variables
    .low		resd	1
    .high		resd	1
    .band		resd	1
    .freq_mod   resd	1
;// stereo variables	
    .low2		resd	1
    .high2		resd	1
    .band2		resd	1	
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // DST structs
; //----------------------------------------------------------------------------------------
GO4K_DST_ID		equ		4
%macro	GO4K_DST 3
    db	%1
%ifdef GO4K_USE_DST_SH	
    db	%2
%endif
%ifdef GO4K_USE_DST_STEREO	
    db	%3
%endif
%endmacro
%define	DRIVE(val)		val
%define	SNHFREQ(val)	val
%define	FLAGS(val)		val
struc	go4kDST_val
;//	unit paramters
    .drive		resd	1
%ifdef GO4K_USE_DST_SH	
    .snhfreq	resd	1
%endif
%ifdef GO4K_USE_DST_STEREO	
    .flags		resd	1
%endif
    .size
endstruc
struc	go4kDST_wrk
;//	work variables
    .out		resd	1
    .snhphase	resd	1
;// stereo variables
    .out2		resd	1
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // DLL structs
; //----------------------------------------------------------------------------------------
GO4K_DLL_ID		equ		5
%macro	GO4K_DLL 8
    db	%1
    db	%2
    db	%3
%ifdef GO4K_USE_DLL_DAMP	
    db	%4
%endif		
%ifdef GO4K_USE_DLL_CHORUS
    db	%5
    db	%6
%endif
    db	%7
    db	%8
%endmacro
%define	PREGAIN(val)	val
%define	DRY(val)		val
%define	FEEDBACK(val)	val
%define	DEPTH(val)		val
%define	DAMP(val)		val
%define	DELAY(val)		val
%define	COUNT(val)		val
struc	go4kDLL_val
;//	unit paramters
    .pregain	resd	1
    .dry		resd	1
    .feedback	resd	1
%ifdef GO4K_USE_DLL_DAMP	
    .damp		resd	1	
%endif
%ifdef GO4K_USE_DLL_CHORUS
    .freq		resd	1
    .depth      resd	1
%endif	
    .delay		resd	1
    .count		resd	1
    .size
endstruc
struc	go4kDLL_wrk
;//	work variables
    .index		resd	1
    .store		resd	1
    .dcin		resd	1
    .dcout		resd	1
    .phase		resd	1
;// the delay buffer	
    .buffer		resd	MAX_DELAY
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // FOP structs
; //----------------------------------------------------------------------------------------
GO4K_FOP_ID	equ			6
%macro	GO4K_FOP 1
    db	%1
%endmacro
%define	OP(val)			val
%define	FOP_POP			0x1
%define	FOP_ADDP		0x2
%define	FOP_MULP		0x3
%define	FOP_PUSH		0x4
%define	FOP_XCH			0x5
%define	FOP_ADD			0x6
%define	FOP_MUL			0x7
%define	FOP_ADDP2		0x8
%define FOP_LOADNOTE	0x9
%define	FOP_MULP2		0xa
struc	go4kFOP_val
    .flags		resd	1
    .size
endstruc
struc	go4kFOP_wrk
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // FST structs
; //----------------------------------------------------------------------------------------
GO4K_FST_ID		equ		7
%macro	GO4K_FST 2
    db	%1
    dw	%2
%endmacro
%define	AMOUNT(val)		val
%define	VALUE_MOD(unit,unittype,slot,flags) unit*MAX_UNIT_SLOTS+go4k %+ unittype %+ _val. %+ slot /4+MAX_WORK_VARS+flags
%define	WRK_MOD(unit,unittype,slot,flags) unit*MAX_UNIT_SLOTS+go4k %+ unittype %+ _wrk. %+ slot /4+flags
%define	FST_SET			0x0000
%define	FST_ADD			0x4000
%define	FST_POP			0x8000
struc	go4kFST_val
    .amount		resd	1
    .size
endstruc
struc	go4kFST_wrk
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // PAN structs
; //----------------------------------------------------------------------------------------
GO4K_PAN_ID		equ		8
%macro	GO4K_PAN 1
%ifdef GO4K_USE_PAN
    db	%1
%endif
%endmacro
%define	PANNING(val)	val
struc	go4kPAN_val
%ifdef GO4K_USE_PAN
    .panning	resd	1
%endif
    .size
endstruc
struc	go4kPAN_wrk
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // OUT structs
; //----------------------------------------------------------------------------------------
GO4K_OUT_ID		equ		9
%macro	GO4K_OUT 2
    db	%1
%ifdef GO4K_USE_GLOBAL_DLL	
    db	%2
%endif	
%endmacro
%define	AUXSEND(val)	val
struc	go4kOUT_val
    .gain		resd	1
%ifdef GO4K_USE_GLOBAL_DLL	
    .auxsend	resd	1
%endif
    .size
endstruc
struc	go4kOUT_wrk
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // ACC structs (this is for the synth	def	only)
; //----------------------------------------------------------------------------------------
GO4K_ACC_ID		equ		10
%macro	GO4K_ACC 1
    db	%1
%endmacro
%define	OUTPUT			0
%define	AUX				8
%define	ACCTYPE(val)	val
struc	go4kACC_val
    .acctype	resd	1
    .size
endstruc
struc	go4kACC_wrk
    .size
endstruc
%ifdef GO4K_USE_FLD
; //----------------------------------------------------------------------------------------
; // FLD structs
; //----------------------------------------------------------------------------------------
GO4K_FLD_ID	equ		11
%macro	GO4K_FLD 1
    db	%1
%endmacro
%define	VALUE(val)	val
struc	go4kFLD_val
    .value		resd	1
    .size
endstruc
struc	go4kFLD_wrk
    .size
endstruc
%endif
%ifdef GO4K_USE_GLITCH
; //----------------------------------------------------------------------------------------
; // GLITCH structs
; //----------------------------------------------------------------------------------------
GO4K_GLITCH_ID		equ		12
%macro	GO4K_GLITCH 5
    db	%1
    db	%2
    db	%3
    db	%4
    db	%5
%endmacro
%define	ACTIVE(val)		val
%define	SLICEFACTOR(val)val
%define	PITCHFACTOR(val)val
%define	SLICESIZE(val)	val
struc	go4kGLITCH_val
;//	unit paramters
    .active		resd	1
    .dry		resd	1
    .dsize		resd	1
    .dpitch		resd	1
    .slicesize	resd	1
    .size
endstruc
struc	go4kGLITCH_wrk
;//	work variables
    .index		resd	1
    .store		resd	1
    .slizesize	resd	1
    .slicepitch	resd	1
    .unused		resd	1
;// the delay buffer	
    .buffer		resd	MAX_DELAY
    .size
endstruc
%endif
%ifdef GO4K_USE_FSTG
; //----------------------------------------------------------------------------------------
; // FSTG structs
; //----------------------------------------------------------------------------------------
%ifdef GO4K_USE_GLITCH
GO4K_FSTG_ID	equ		13
%else
GO4K_FSTG_ID	equ		12
%endif
%macro	GO4K_FSTG 2
    db	%1
    dw	%2
%endmacro

%define	GLOBAL_VALUE_MOD(inst,unit,unittype,slot,flags) inst*go4k_instrument.size*MAX_VOICES/4 + unit*MAX_UNIT_SLOTS+go4k %+ unittype %+ _val. %+ slot /4+(go4k_instrument.workspace/4)+MAX_WORK_VARS+flags
%define	GLOBAL_WRK_MOD(inst,unit,unittype,slot,flags) inst*go4k_instrument.size*MAX_VOICES/4 + unit*MAX_UNIT_SLOTS+go4k %+ unittype %+ _wrk. %+ slot /4+(go4k_instrument.workspace/4)+flags
struc	go4kFSTG_val
    .amount		resd	1
    .size
endstruc
struc	go4kFSTG_wrk
    .size
endstruc
%endif
; //----------------------------------------------------------------------------------------
; // Voice struct
; //----------------------------------------------------------------------------------------
struc	go4k_instrument
    .release	resd	1
    .note		resd	1
    .workspace	resd	MAX_UNITS*MAX_UNIT_SLOTS
    .dlloutl	resd	1
    .dlloutr	resd	1
    .outl		resd	1
    .outr		resd	1
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // Synth struct
; //----------------------------------------------------------------------------------------
struc	go4k_synth
    .instruments	resb	go4k_instrument.size * MAX_INSTRUMENTS * MAX_VOICES
    .global			resb	go4k_instrument.size * MAX_VOICES
    .size
endstruc

%endif ; _4KLANG_INC