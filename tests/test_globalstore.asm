%define	MAX_INSTRUMENTS	1
%define	BPM	100
%define	MAX_PATTERNS 1
%define	SINGLE_FILE
%define USE_SECTIONS
%define GO4K_USE_FLD
%define GO4K_USE_FLD_MOD_VM
%define GO4K_USE_FSTG

%include "../src/4klang.asm"

; //-------------------------------------------------------------------------------
; // Pattern Data
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc1)

EXPORT MANGLE_DATA(go4k_patterns)
	db 64, HLD, HLD, HLD, HLD, HLD, HLD, HLD,	0, 0, 0, 0,	0, 0, 0, 0,		

; //----------------------------------------------------------------------------------------
; // Pattern Index List
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc2)

EXPORT MANGLE_DATA(go4k_pattern_lists)
Instrument0List		db	0,

; //----------------------------------------------------------------------------------------
; // Instrument	Commands
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc3)

EXPORT MANGLE_DATA(go4k_synth_instructions)
GO4K_BEGIN_CMDDEF(Instrument0)
	db GO4K_FLD_ID	
	db GO4K_FSTG_ID
	db GO4K_FLD_ID	
	db GO4K_FLD_ID		
	db GO4K_OUT_ID
GO4K_END_CMDDEF
;//	global commands
GO4K_BEGIN_CMDDEF(Global)	
	db GO4K_ACC_ID
	db GO4K_FLD_ID	
	db GO4K_FSTG_ID
	db GO4K_FLD_ID	
	db GO4K_FOP_ID			
	db GO4K_OUT_ID
GO4K_END_CMDDEF

; //----------------------------------------------------------------------------------------
; // Intrument Data
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc4)

EXPORT MANGLE_DATA(go4k_synth_parameter_values)
GO4K_BEGIN_PARAMDEF(Instrument0)	
	GO4K_FLD	VALUE(128)
	GO4K_FSTG	AMOUNT(96),DEST((1*go4k_instrument.size*MAX_VOICES/4)+(3*MAX_UNIT_SLOTS+0)+(go4k_instrument.workspace/4)+FST_SET+FST_POP)	
	GO4K_FLD	VALUE(64)
	GO4K_FLD	VALUE(64)
	GO4K_OUT	GAIN(128), AUXSEND(0)
GO4K_END_PARAMDEF
;//	global parameters
GO4K_BEGIN_PARAMDEF(Global)	
	GO4K_ACC	ACCTYPE(OUTPUT)	
	GO4K_FLD	VALUE(0)
	GO4K_FSTG	AMOUNT(96),DEST((0*go4k_instrument.size*MAX_VOICES/4)+(2*MAX_UNIT_SLOTS+0)+(go4k_instrument.workspace/4)+FST_SET+FST_POP)
	GO4K_FLD	VALUE(64)
	GO4K_FOP	OP(FOP_ADDP)
	GO4K_OUT	GAIN(128), AUXSEND(0)
GO4K_END_PARAMDEF

; //----------------------------------------------------------------------------------------
; // Export MAX_SAMPLES for test_renderer
; //----------------------------------------------------------------------------------------
SECT_DATA(g4krender)

EXPORT MANGLE_DATA(test_max_samples)
	dd MAX_SAMPLES