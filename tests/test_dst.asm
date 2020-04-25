%define	MAX_INSTRUMENTS	1
%define	BPM	100
%define	MAX_PATTERNS 1
%define	SINGLE_FILE
%define USE_SECTIONS
%define GO4K_USE_DST
 
%include "../src/4klang.asm"

; //----------------------------------------------------------------------------------------
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
	db GO4K_ENV_ID	
	db GO4K_DST_ID	
	db GO4K_ENV_ID	
	db GO4K_DST_ID	
	db GO4K_OUT_ID
GO4K_END_CMDDEF
;//	global commands
GO4K_BEGIN_CMDDEF(Global)	
	db GO4K_ACC_ID	
	db GO4K_OUT_ID
GO4K_END_CMDDEF
go4k_synth_instructions_end
; //----------------------------------------------------------------------------------------
; // Intrument Data
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc4)

EXPORT MANGLE_DATA(go4k_synth_parameter_values)
GO4K_BEGIN_PARAMDEF(Instrument0)
	GO4K_ENV	ATTAC(64),DECAY(64),SUSTAIN(64),RELEASE(80),GAIN(128)	
	GO4K_DST	DRIVE(32), SNHFREQ(128), FLAGS(0)
	GO4K_ENV	ATTAC(64),DECAY(64),SUSTAIN(64),RELEASE(80),GAIN(128)	
	GO4K_DST	DRIVE(96), SNHFREQ(128), FLAGS(0)
	GO4K_OUT	GAIN(128), AUXSEND(0)
GO4K_END_PARAMDEF
;//	global parameters
GO4K_BEGIN_PARAMDEF(Global)	
	GO4K_ACC	ACCTYPE(OUTPUT)	
	GO4K_OUT	GAIN(128), AUXSEND(0)
GO4K_END_PARAMDEF

; //----------------------------------------------------------------------------------------
; // Export MAX_SAMPLES for test_renderer
; //----------------------------------------------------------------------------------------
SECT_DATA(g4krender)

EXPORT MANGLE_DATA(test_max_samples)
	dd MAX_SAMPLES