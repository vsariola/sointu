%define	MAX_INSTRUMENTS	9
%define	MAX_VOICES 1
%define	HLD	1 ;	// can be adjusted to give crinkler	some other possibilities
%define	BPM	100
%define	MAX_PATTERNS 62
%define	SINGLE_FILE
%define USE_SECTIONS
%define	GO4K_USE_UNDENORMALIZE			; // removing this skips denormalization code in the units
%define	GO4K_CLIP_OUTPUT				; // removing this skips clipping code for the final output
%define	GO4K_USE_DST					; // removing this will	skip DST unit
%define	GO4K_USE_DLL					; // removing this will	skip DLL unit
%define	GO4K_USE_PAN					; // removing this will	skip PAN unit
%define	GO4K_USE_GLOBAL_DLL				; // removing this will	skip global	dll	processing
%define	GO4K_USE_FSTG					; // removing this will	skip global	store unit
%define	GO4K_USE_FLD					; // removing this will	skip float load unit
%define	GO4K_USE_GLITCH					; // removing this will	skip GLITCH unit
%define	GO4K_USE_ENV_CHECK				; // removing this skips checks	if processing is needed
%define	GO4K_USE_ENV_MOD_GM				; // removing this will	skip env gain modulation code
%define	GO4K_USE_ENV_MOD_ADR			; // removing this will skip env attack/decay/release modulation code
%define	GO4K_USE_VCO_CHECK				; // removing this skips checks	if processing is needed
%define	GO4K_USE_VCO_PHASE_OFFSET		; // removing this will	skip initial phase offset code
%define	GO4K_USE_VCO_SHAPE				; // removing this skips waveshaping code
%define	GO4K_USE_VCO_GATE				; // removing this skips gate code
%define	GO4K_USE_VCO_MOD_FM				; // removing this skips frequency modulation code
%define	GO4K_USE_VCO_MOD_PM				; // removing this skips phase modulation code
%define	GO4K_USE_VCO_MOD_TM				; // removing this skips transpose modulation code
%define	GO4K_USE_VCO_MOD_DM				; // removing this skips detune	modulation code
%define	GO4K_USE_VCO_MOD_CM				; // removing this skips color modulation code
%define	GO4K_USE_VCO_MOD_GM				; // removing this skips gain modulation code
%define	GO4K_USE_VCO_MOD_SM				; // removing this skips shaping modulation	code
%define	GO4K_USE_VCO_STEREO				; // removing this skips stereo code
%define	GO4K_USE_VCF_CHECK				; // removing this skips checks	if processing is needed
%define	GO4K_USE_VCF_MOD_FM				; // removing this skips frequency modulation code
%define	GO4K_USE_VCF_MOD_RM				; // removing this skips resonance modulation code
%define	GO4K_USE_VCF_HIGH				; // removing this skips code for high output
%define	GO4K_USE_VCF_BAND				; // removing this skips code for band output
%define	GO4K_USE_VCF_PEAK				; // removing this skips code for peak output
%define	GO4K_USE_VCF_STEREO				; // removing this skips code for stereo filter output
%define	GO4K_USE_DST_CHECK				; // removing this skips checks	if processing is needed
%define	GO4K_USE_DST_SH					; // removing this skips sample	and	hold code
%define	GO4K_USE_DST_MOD_DM				; // removing this skips distortion	modulation code
%define	GO4K_USE_DST_MOD_SH				; // removing this skips sample	and	hold modulation	code
%define	GO4K_USE_DST_STEREO				; // removing this skips stereo processing
%define	GO4K_USE_DLL_NOTE_SYNC			; // removing this will	skip delay length adjusting	code (karplus strong)
%define	GO4K_USE_DLL_CHORUS				; // removing this will	skip delay chorus/flanger code
%define	GO4K_USE_DLL_CHORUS_CLAMP		; // removing this will skip chorus lfo phase clamping
%define	GO4K_USE_DLL_DAMP				; // removing this will	skip dll damping code
%define	GO4K_USE_DLL_DC_FILTER			; // removing this will	skip dll dc	offset removal code
%define	GO4K_USE_FSTG_CHECK				; // removing this skips checks	if processing is needed
%define	GO4K_USE_PAN_MOD				; // removing this will	skip panning modulation	code
%define	GO4K_USE_OUT_MOD_AM				; // removing this skips output	aux	send modulation	code
%define	GO4K_USE_OUT_MOD_GM				; // removing this skips output	gain modulation	code
%define	GO4K_USE_WAVESHAPER_CLIP		; // removing this will	skip clipping code
%define	GO4K_USE_FLD_MOD_VM				; // removing this will	skip float load modulation	code
%define GO4K_USE_DLL_MOD				; // define this to enable modulations for delay line
%define GO4K_USE_DLL_MOD_PM				; // define this to enable pregain modulation for delay line
%define GO4K_USE_DLL_MOD_FM				; // define this to enable feebback modulation for delay line
%define GO4K_USE_DLL_MOD_IM				; // define this to enable dry modulation for delay line
%define GO4K_USE_DLL_MOD_DM				; // define this to enable damping modulation for delay line
%define GO4K_USE_DLL_MOD_SM				; // define this to enable lfo freq modulation for delay line
%define GO4K_USE_DLL_MOD_AM				; // define this to enable lfo depth modulation for delay line

%include "../4klang.asm"

; //----------------------------------------------------------------------------------------
; // Pattern Data, reduced by 967 patterns
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc1)

EXPORT MANGLE_DATA(go4k_patterns)
	db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	
	db	76,	HLD, HLD, HLD, 0, 0, 0,	0, 79, HLD,	HLD, HLD, 0, 0,	0, 0, 
	db	69,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 0, 0, 0,	0, 0, 0, 0,	0, 
	db	76,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	81,	HLD, HLD, HLD, 0, 0, 0,	0, 79, HLD,	HLD, HLD, 0, 0,	0, 0, 
	db	84,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 0, 0, 0,	0, 0, 0, 0,	0, 
	db	76,	HLD, HLD, HLD, 0, 0, 0,	0, 88, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, 
	db	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	0, 0, 0, 0,	
	db	76,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 0, 0, 0,	0, 0, 0, 0,	0, 
	db	52,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	57,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	60,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	64,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	69,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	72,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	40,	52,	HLD, 64, HLD, 40, 52, 64, 52, HLD, HLD,	HLD, 0,	0, 0, 0, 
	db	57,	0, 0, 57, 0, 0,	69,	0, 0, 69, 0, 0,	57,	HLD, HLD, HLD, 
	db	52,	64,	0, 57, 0, 69, 48, 0, 48, HLD, HLD, HLD,	0, 0, 0, 0,	
	db	40,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 
	db	40,	52,	HLD, 40, HLD, 40, 40, 40, 52, HLD, HLD,	HLD, 0,	0, 0, 0, 
	db	40,	0, 0, 52, 0, 0,	40,	0, 52, 40, 0, 52, 0, 40, 0,	0, 
	db	45,	0, 0, 57, 0, 0,	45,	0, 57, 45, 0, 57, 0, 45, 0,	0, 
	db	48,	0, 0, 60, 0, 0,	48,	0, 60, 48, 0, 60, 0, 48, 0,	0, 
	db	40,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 0, 0, 0,	0, 
	db	45,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 0, 0, 0,	0, 
	db	36,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, 0, 0, 0,	0, 
	db	0, 0, 0, 0,	60,	HLD, 0,	0, 0, 0, 0,	0, 60, HLD,	0, 0, 
	db	60,	HLD, 0,	0, 0, 0, 60, HLD, 0, 0,	60,	HLD, 60, HLD, 0, 0,	
	db	0, 0, 0, 0,	60,	HLD, 0,	0, 0, 0, 0,	0, 60, HLD,	60,	HLD, 
	db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	60,	HLD, 0,	0, 
	db	60,	HLD, 60, HLD, 0, 0,	60,	HLD, 0,	0, 0, 0, 60, HLD, 0, 0,	
	db	0, 0, 60, HLD, 0, 0, 0,	0, 60, HLD,	0, 0, 0, 0,	0, 0, 
	db	0, 0, 60, HLD, 0, 0, 0,	0, 60, HLD,	0, 0, 60, HLD, 0, 0, 
	db	0, 0, 0, 60, HLD, 0, 0,	0, 0, 0, 0,	0, 60, HLD,	0, 0, 
	db	0, 0, 0, 60, HLD, 0, 0,	0, 0, 60, HLD, 60, 60, HLD,	0, 0, 
	db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	84,	0, 0, 0, 
	db	91,	0, 0, 88, 0, 0,	76,	0, 81, 0, 0, 0,	0, 0, 0, 0,	
	db	81,	0, 0, 84, 0, 0,	86,	0, 88, 0, 0, 0,	0, 0, 0, 0,	
	db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	93,	0, 0, 0, 
	db	81,	0, 0, 84, 0, 0,	86,	0, 81, 0, 0, 0,	0, 0, 0, 0,	
	db	84,	0, 0, 86, 0, 0,	88,	0, 0, 91, 0, 0,	84,	0, 0, 0, 
	db	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	HLD, HLD, HLD, HLD,	
go4k_patterns_end
; //----------------------------------------------------------------------------------------
; // Pattern Index List
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc2)

EXPORT MANGLE_DATA(go4k_pattern_lists)
Instrument0List		db	0, 0, 0, 0,	0, 0, 0, 0,	1, 2, 3, 0,	4, 5, 6, 7,	0, 0, 0, 0,	0, 0, 0, 0,	1, 2, 3, 0,	4, 5, 6, 7,	0, 0, 0, 0,	0, 0, 0, 0,	1, 2, 3, 0,	4, 5, 6, 7,	1, 2, 3, 0,	4, 5, 6, 7,	8, 2, 9, 0,	0, 0,
Instrument1List		db	9, 7, 10, 7, 11, 7,	12,	7, 9, 7, 10, 7,	11,	7, 12, 7, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 12, 7, 13, 7,	14,	7, 3, 7, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 12, 7, 0, 0,
Instrument2List		db	15,	0, 16, 0, 17, 0, 18, 0,	19,	0, 16, 0, 17, 0, 18, 0,	20,	20,	21,	21,	22,	22,	20,	20,	20,	20,	21,	21,	22,	22,	20,	20,	23,	15,	24,	16,	25,	17,	18,	0, 20, 20, 21, 21, 22, 22, 20, 20, 20, 20, 21, 21, 22, 22, 20, 20, 20, 20, 0, 0, 0,	0,
Instrument3List		db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	0, 0, 0, 0,	0, 0, 0, 0,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	26,	0, 0, 0, 0,	0,
Instrument4List		db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	27,	26,	27,	26,	27,	26,	27,	28,	27,	26,	27,	26,	27,	26,	27,	28,	27,	29,	27,	29,	27,	29,	27,	29,	27,	26,	27,	26,	27,	26,	27,	28,	27,	26,	27,	26,	27,	26,	27,	28,	27,	26,	27,	0, 0, 0,
Instrument5List		db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	30,	31,	30,	32,	30,	31,	30,	32,	30,	31,	30,	32,	30,	31,	30,	32,	30,	31,	30,	32,	30,	31,	30,	0, 30, 31, 30, 32, 30, 31, 30, 32, 30, 31, 30, 32, 30, 31, 30, 32, 30, 31, 30, 32, 0, 0,
Instrument6List		db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	33,	34,	33,	33,	0, 0, 0, 0,
Instrument7List		db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 35, 36, 0,	37,	38,	36,	0, 39, 35, 36, 0, 37, 38, 40, 0, 0,	0, 0, 0, 0,	0, 0, 0,
Instrument8List		db	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	0, 0, 0, 0,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	11,	41,	41,	41,	41,	0,
go4k_pattern_lists_end
; //----------------------------------------------------------------------------------------
; // Instrument	Commands
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc3)

EXPORT MANGLE_DATA(go4k_synth_instructions)
GO4K_BEGIN_CMDDEF(Instrument0)
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_FST_ID
	db GO4K_FST_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_VCF_ID
	db GO4K_VCF_ID
	db GO4K_DST_ID
	db GO4K_PAN_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument1)
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_FST_ID
	db GO4K_FST_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_VCF_ID
	db GO4K_VCF_ID
	db GO4K_DST_ID
	db GO4K_PAN_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument2)
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_FST_ID
	db GO4K_FST_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_VCF_ID
	db GO4K_VCF_ID
	db GO4K_DST_ID
	db GO4K_PAN_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument3)
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_VCO_ID
	db GO4K_FOP_ID
	db GO4K_VCF_ID
	db GO4K_PAN_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument4)
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_ENV_ID
	db GO4K_DST_ID
	db GO4K_FST_ID
	db GO4K_FOP_ID
	db GO4K_VCO_ID
	db GO4K_FOP_ID
	db GO4K_PAN_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument5)
	db GO4K_ENV_ID
	db GO4K_VCO_ID
	db GO4K_FOP_ID
	db GO4K_VCF_ID
	db GO4K_PAN_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument6)
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_ENV_ID
	db GO4K_FST_ID
	db GO4K_FST_ID
	db GO4K_FOP_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_VCO_ID
	db GO4K_VCF_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_FOP_ID
	db GO4K_VCF_ID
	db GO4K_PAN_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument7)
	db GO4K_ENV_ID
	db GO4K_VCO_ID
	db GO4K_FOP_ID
	db GO4K_PAN_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
GO4K_BEGIN_CMDDEF(Instrument8)
	db GO4K_ENV_ID
	db GO4K_VCO_ID
	db GO4K_FOP_ID
	db GO4K_FSTG_ID
	db GO4K_FSTG_ID
	db GO4K_FOP_ID
GO4K_END_CMDDEF
;//	global commands
GO4K_BEGIN_CMDDEF(Global)
	db GO4K_ACC_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_DLL_ID
	db GO4K_FOP_ID
	db GO4K_ACC_ID
	db GO4K_FOP_ID
	db GO4K_OUT_ID
GO4K_END_CMDDEF
go4k_synth_instructions_end
; //----------------------------------------------------------------------------------------
; // Intrument Data
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc4)

EXPORT MANGLE_DATA(go4k_synth_parameter_values)
GO4K_BEGIN_PARAMDEF(Instrument0)
	GO4K_ENV	ATTAC(72),DECAY(96),SUSTAIN(96),RELEASE(88),GAIN(128)
;	GO4K_FST	AMOUNT(64),DEST(0*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_VCO	TRANSPOSE(64),DETUNE(60),PHASE(32),GATES(0),COLOR(80),SHAPE(64),GAIN(128),FLAGS(PULSE)
	GO4K_VCO	TRANSPOSE(64),DETUNE(72),PHASE(32),GATES(0),COLOR(96),SHAPE(64),GAIN(128),FLAGS(TRISAW)
	GO4K_VCO	TRANSPOSE(32),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(96),GAIN(128),FLAGS(SINE|LFO)
; 	GO4K_FST	AMOUNT(68),DEST(2*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
;	GO4K_FST	AMOUNT(61),DEST(3*MAX_UNIT_SLOTS+2); TODO: convert into new DEST format
	GO4K_FOP	OP(FOP_POP)
	GO4K_FOP	OP(FOP_ADDP)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_VCF	FREQUENCY(26),RESONANCE(128),VCFTYPE(PEAK)
	GO4K_VCF	FREQUENCY(64),RESONANCE(64),VCFTYPE(LOWPASS)
	GO4K_DST	DRIVE(104),	SNHFREQ(128), FLAGS(0)
	GO4K_PAN	PANNING(64)
	GO4K_DLL	PREGAIN(96),DRY(128),FEEDBACK(96),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(16),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_DLL	PREGAIN(96),DRY(128),FEEDBACK(64),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(17),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_OUT	GAIN(0), AUXSEND(32)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument1)
	GO4K_ENV	ATTAC(72),DECAY(96),SUSTAIN(96),RELEASE(88),GAIN(128)
;	GO4K_FST	AMOUNT(64),DEST(0*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_VCO	TRANSPOSE(64),DETUNE(60),PHASE(32),GATES(0),COLOR(80),SHAPE(64),GAIN(128),FLAGS(TRISAW)
	GO4K_VCO	TRANSPOSE(64),DETUNE(72),PHASE(32),GATES(0),COLOR(96),SHAPE(112),GAIN(64),FLAGS(SINE)
	GO4K_VCO	TRANSPOSE(80),DETUNE(112),PHASE(0),GATES(0),COLOR(64),SHAPE(16),GAIN(128),FLAGS(PULSE|LFO)
;	GO4K_FST	AMOUNT(68),DEST(2*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
;	GO4K_FST	AMOUNT(60),DEST(3*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_FOP	OP(FOP_POP)
	GO4K_FOP	OP(FOP_ADDP)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_VCF	FREQUENCY(80),RESONANCE(24),VCFTYPE(LOWPASS)
	GO4K_VCF	FREQUENCY(48),RESONANCE(24),VCFTYPE(HIGHPASS)
	GO4K_DST	DRIVE(64), SNHFREQ(128), FLAGS(0)
	GO4K_PAN	PANNING(64)
	GO4K_DLL	PREGAIN(96),DRY(128),FEEDBACK(96),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(16),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_DLL	PREGAIN(96),DRY(128),FEEDBACK(64),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(17),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_OUT	GAIN(0), AUXSEND(32)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument2)
	GO4K_ENV	ATTAC(32),DECAY(64),SUSTAIN(64),RELEASE(64),GAIN(64)
;	GO4K_FST	AMOUNT(120),DEST(0*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_VCO	TRANSPOSE(64),DETUNE(64),PHASE(32),GATES(0),COLOR(80),SHAPE(64),GAIN(128),FLAGS(PULSE)
	GO4K_VCO	TRANSPOSE(64),DETUNE(72),PHASE(32),GATES(0),COLOR(96),SHAPE(64),GAIN(128),FLAGS(TRISAW)
	GO4K_VCO	TRANSPOSE(32),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(96),GAIN(128),FLAGS(SINE|LFO)
;	GO4K_FST	AMOUNT(68),DEST(2*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
;	GO4K_FST	AMOUNT(60),DEST(3*MAX_UNIT_SLOTS+2); TODO: convert into new DEST format
	GO4K_FOP	OP(FOP_POP)
	GO4K_FOP	OP(FOP_ADDP)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_VCF	FREQUENCY(18),RESONANCE(64),VCFTYPE(PEAK)
	GO4K_VCF	FREQUENCY(32),RESONANCE(48),VCFTYPE(LOWPASS)
	GO4K_DST	DRIVE(88), SNHFREQ(128), FLAGS(0)
	GO4K_PAN	PANNING(64)
	GO4K_DLL	PREGAIN(64),DRY(128),FEEDBACK(96),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(16),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_DLL	PREGAIN(64),DRY(128),FEEDBACK(64),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(17),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_OUT	GAIN(64), AUXSEND(64)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument3)
	GO4K_ENV	ATTAC(0),DECAY(76),SUSTAIN(0),RELEASE(0),GAIN(32)
;	GO4K_FST	AMOUNT(128),DEST(0*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_VCO	TRANSPOSE(64),DETUNE(64),PHASE(64),GATES(0),COLOR(64),SHAPE(64),GAIN(128),FLAGS(NOISE)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_VCF	FREQUENCY(80),RESONANCE(128),VCFTYPE(LOWPASS)
	GO4K_PAN	PANNING(64)
	GO4K_OUT	GAIN(64), AUXSEND(0)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument4)
	GO4K_ENV	ATTAC(0),DECAY(64),SUSTAIN(96),RELEASE(64),GAIN(128)
;	GO4K_FST	AMOUNT(128),DEST(0*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_ENV	ATTAC(0),DECAY(70),SUSTAIN(0),RELEASE(0),GAIN(128)
	GO4K_DST	DRIVE(32), SNHFREQ(128), FLAGS(0)
;	GO4K_FST	AMOUNT(80),DEST(6*MAX_UNIT_SLOTS+1) ; TODO: convert into new DEST format
	GO4K_FOP	OP(FOP_POP)
	GO4K_VCO	TRANSPOSE(46),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(64),GAIN(128),FLAGS(TRISAW)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_PAN	PANNING(64)
	GO4K_OUT	GAIN(128), AUXSEND(0)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument5)
	GO4K_ENV	ATTAC(0),DECAY(64),SUSTAIN(0),RELEASE(0),GAIN(128)
	GO4K_VCO	TRANSPOSE(64),DETUNE(64),PHASE(64),GATES(0),COLOR(64),SHAPE(64),GAIN(128),FLAGS(NOISE)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_VCF	FREQUENCY(128),RESONANCE(128),VCFTYPE(BANDPASS)
	GO4K_PAN	PANNING(64)
	GO4K_DLL	PREGAIN(64),DRY(128),FEEDBACK(96),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(16),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_DLL	PREGAIN(64),DRY(128),FEEDBACK(64),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(17),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_OUT	GAIN(64), AUXSEND(0)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument6)
	GO4K_ENV	ATTAC(0),DECAY(72),SUSTAIN(0),RELEASE(72),GAIN(128)
;	GO4K_FST	AMOUNT(128),DEST(0*MAX_UNIT_SLOTS+2) ; TODO: convert into new DEST format
	GO4K_ENV	ATTAC(0),DECAY(56),SUSTAIN(0),RELEASE(0),GAIN(128)
;	GO4K_FST	AMOUNT(108),DEST(6*MAX_UNIT_SLOTS+1) ; TODO: convert into new DEST format
;	GO4K_FST	AMOUNT(72),DEST(7*MAX_UNIT_SLOTS+1) ; TODO: convert into new DEST format
	GO4K_FOP	OP(FOP_POP)
	GO4K_VCO	TRANSPOSE(32),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(32),GAIN(64),FLAGS(SINE)
	GO4K_VCO	TRANSPOSE(64),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(80),GAIN(64),FLAGS(SINE)
	GO4K_VCO	TRANSPOSE(64),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(64),GAIN(64),FLAGS(NOISE)
	GO4K_VCF	FREQUENCY(104),RESONANCE(128),VCFTYPE(LOWPASS)
	GO4K_FOP	OP(FOP_ADDP)
	GO4K_FOP	OP(FOP_ADDP)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_VCF	FREQUENCY(22),RESONANCE(32),VCFTYPE(HIGHPASS)
	GO4K_PAN	PANNING(64)
	GO4K_OUT	GAIN(64), AUXSEND(0)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument7)
	GO4K_ENV	ATTAC(0),DECAY(0),SUSTAIN(96),RELEASE(32),GAIN(128)
	GO4K_VCO	TRANSPOSE(64),DETUNE(64),PHASE(0),GATES(0),COLOR(80),SHAPE(64),GAIN(128),FLAGS(PULSE)
	GO4K_FOP	OP(FOP_MULP)
	GO4K_PAN	PANNING(64)
	GO4K_DLL	PREGAIN(96),DRY(128),FEEDBACK(96),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(16),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_DLL	PREGAIN(96),DRY(128),FEEDBACK(64),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(17),COUNT(1)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_OUT	GAIN(0), AUXSEND(64)
GO4K_END_PARAMDEF
GO4K_BEGIN_PARAMDEF(Instrument8)
	GO4K_ENV	ATTAC(0),DECAY(0),SUSTAIN(128),RELEASE(0),GAIN(128)
	GO4K_VCO	TRANSPOSE(48),DETUNE(64),PHASE(0),GATES(0),COLOR(64),SHAPE(64),GAIN(128),FLAGS(TRISAW|LFO)
	GO4K_FOP	OP(FOP_MULP)
;	GO4K_FSTG	AMOUNT(72),DEST(2*go4k_instrument.size*MAX_VOICES+10*MAX_UNIT_SLOTS*4+4*4+go4k_instrument.workspace) ; TODO: convert into new DEST format
;	GO4K_FSTG	AMOUNT(66),DEST(1*go4k_instrument.size*MAX_VOICES+10*MAX_UNIT_SLOTS*4+4*4+go4k_instrument.workspace) ; TODO: convert into new DEST format
	GO4K_FOP	OP(FOP_POP)
GO4K_END_PARAMDEF
;//	global parameters
GO4K_BEGIN_PARAMDEF(Global)
	GO4K_ACC	ACCTYPE(AUX)
	GO4K_DLL	PREGAIN(40),DRY(128),FEEDBACK(125),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(0),COUNT(8)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_DLL	PREGAIN(40),DRY(128),FEEDBACK(125),DAMP(64),FREQUENCY(0),DEPTH(0),DELAY(8),COUNT(8)
	GO4K_FOP	OP(FOP_XCH)
	GO4K_ACC	ACCTYPE(OUTPUT)
	GO4K_FOP	OP(FOP_ADDP2)
	GO4K_OUT	GAIN(64), AUXSEND(0)
GO4K_END_PARAMDEF
go4k_synth_parameter_values_end
; //----------------------------------------------------------------------------------------
; // Delay/Reverb Times
; //----------------------------------------------------------------------------------------
SECT_DATA(g4kmuc5)

EXPORT MANGLE_DATA(go4k_delay_times)
	dw 0
	dw 1116
	dw 1188
	dw 1276
	dw 1356
	dw 1422
	dw 1492
	dw 1556
	dw 1618
	dw 1140
	dw 1212
	dw 1300
	dw 1380
	dw 1446
	dw 1516
	dw 1580
	dw 1642
	dw 22050		; Originally times 100 dw 0, but crashes for me (Peter) - so reverted to this found in an older version
	dw 16537
	dw 11025