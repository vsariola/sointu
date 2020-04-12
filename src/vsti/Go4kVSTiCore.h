#include "windows.h"

// init synth
void Go4kVSTi_Init();

// clear instrument slot
void Go4kVSTi_ClearInstrumentSlot(char channel, int slot);
// clear instrument workspace
void Go4kVSTi_ClearInstrumentWorkspace(char channel);
// clear global slot
void Go4kVSTi_ClearGlobalSlot(int slot);
// clear global workspace
void Go4kVSTi_ClearGlobalWorkspace();

// reset instrument to default values
void Go4kVSTi_ResetInstrument(char channel);
// reset global to default values
void Go4kVSTi_ResetGlobal();
// reset the synth to default values
void Go4kVSTi_ResetPatch();

// flip 2 neighbour instrument slots
void Go4kVSTi_FlipInstrumentSlots(char channel, int a, int b);
// flip 2 neighbour global slots
void Go4kVSTi_FlipGlobalSlots(int a, int b);

// init a slot with given type with default values
void Go4kVSTi_InitSlot(BYTE* slot, int channel, int type);
// init a instrument slot
void Go4kVSTi_InitInstrumentSlot(char channel, int s, int type);
// init a global slot
void Go4kVSTi_InitGlobalSlot(int s, int type);

// set global bpm
void Go4kVSTi_SetBPM(float bpm);
float Go4kVSTi_GetBPM();

// activate solo mode
void Go4kVSTi_Solo(int channel, int solo);
// stream recording activation/deactivation
void Go4kVSTi_Record(bool record, bool recordingNoise, int patternsize, float patternquant);
// panic
void Go4kVSTi_Panic();
// update dll times (e.g. sync to current bpm)
void Go4kVSTi_UpdateDelayTimes();
// clear delay lines
void Go4kVSTi_ClearDelayLines();

// one tick the whole synth pipeline. results are left and right output sample
void Go4kVSTi_Tick(float *oleft, float *oright, int samples);
// add a voice with given parameters to synth
void Go4kVSTi_AddVoice(int channel, int note);
// stop a voice with given parameters in synth
void Go4kVSTi_StopVoice(int channel, int note);

// load binary patch data
void Go4kVSTi_LoadPatch(char *filename);
// save binary patch data
void Go4kVSTi_SavePatch(char *filename);
// load instrumen data to specified channel
void Go4kVSTi_LoadInstrument(char* filename, char channel);
// save instrument data from current channel
void Go4kVSTi_SaveInstrument(char* filename, char channel);
// load unit data into specified slot
void Go4kVSTi_LoadUnit(char* filename, BYTE* slot);
// save unit date from specified slot
void Go4kVSTi_SaveUnit(char* filename, BYTE* slot);

void Go4kVSTi_SaveByteStream(HINSTANCE hInst, char* filename, int useenvlevels, int useenotevalues, int clipoutput, int undenormalize, int objformat, int output16);

#define MAX_POLYPHONY		2
#define MAX_INSTRUMENTS		16
#define MAX_UNITS			64
#define MAX_UNIT_SLOTS		16

enum UnitID
{
	M_NONE = 0,
	M_ENV,
	M_VCO,
	M_VCF,
	M_DST,
	M_DLL,
	M_FOP,
	M_FST,
	M_PAN,
	M_OUT,
	M_ACC,
	M_FLD,
	M_GLITCH,
	NUM_MODULES
};

/////////////////////////////////////////////////////////////////////////////////////
// value definitions
/////////////////////////////////////////////////////////////////////////////////////


#define ENV_STATE_ATTAC		0
#define ENV_STATE_DECAY		1
#define ENV_STATE_SUSTAIN	2
#define ENV_STATE_RELEASE	3
#define ENV_STATE_OFF		4

typedef struct ENV_val
{
	BYTE	id;
	BYTE	attac;
	BYTE	decay;
	BYTE	sustain;
	BYTE	release;
	BYTE	gain;
// GUI STUFF
} *ENV_valP;

#define VCO_SINE		0x01
#define VCO_TRISAW		0x02
#define VCO_PULSE		0x04
#define VCO_NOISE		0x08
#define VCO_LFO			0x10
#define VCO_GATE		0x20
#define	VCO_STEREO		0x40
typedef struct VCO_val
{
	BYTE	id;
	BYTE	transpose;
	BYTE	detune;
	BYTE	phaseofs;
	BYTE	gate;
	BYTE	color;
	BYTE	shape;
	BYTE	gain;
	BYTE	flags;
// GUI STUFF
} *VCO_valP;

typedef struct VCO11_val
{
	BYTE	id;
	BYTE	transpose;
	BYTE	detune;
	BYTE	phaseofs;
	BYTE	color;
	BYTE	shape;
	BYTE	gain;
	BYTE	flags;
// GUI STUFF
} *VCO11_valP;

#define VCF_LOWPASS		0x1
#define VCF_HIGHPASS	0x2
#define VCF_BANDPASS	0x4
#define	VCF_BANDSTOP	0x3
#define VCF_ALLPASS		0x7
#define	VCF_PEAK		0x8
#define	VCF_STEREO		0x10
typedef struct VCF_val
{
	BYTE	id;
	BYTE	freq;
	BYTE	res;
	BYTE	type;
// GUI STUFF
} *VCF_valP;

typedef struct DST_val
{
	BYTE	id;
	BYTE	drive;
	BYTE	snhfreq;
	BYTE	stereo;
// GUI STUFF
} *DST_valP;

typedef struct DLL_val
{
	BYTE	id;
	BYTE	pregain;
	BYTE	dry;
	BYTE	feedback;
	BYTE	damp;
	BYTE	freq;
	BYTE	depth;
	BYTE	delay;
	BYTE	count;
// GUI STUFF
	BYTE	guidelay;
	BYTE	synctype;
	BYTE	leftreverb;
	BYTE	reverb;
} *DLL_valP;

typedef struct DLL10_val
{
	BYTE	id;
	BYTE	pregain;
	BYTE	dry;
	BYTE	feedback;
	BYTE	damp;
	BYTE	delay;
	BYTE	count;
// GUI STUFF
	BYTE	guidelay;
	BYTE	synctype;
	BYTE	leftreverb;
	BYTE	reverb;
} *DLL10_valP;

#define FOP_POP			0x1
#define FOP_ADDP		0x2
#define FOP_MULP		0x3
#define FOP_PUSH		0x4
#define FOP_XCH			0x5
#define FOP_ADD			0x6
#define FOP_MUL			0x7
#define FOP_ADDP2		0x8
#define FOP_LOADNOTE	0x9
#define FOP_MULP2		0xa
typedef struct FOP_val
{
	BYTE	id;
	BYTE	flags;
} *FOP_valP;

#define	FST_SET			0x00
#define	FST_ADD			0x10
#define	FST_MUL			0x20
#define	FST_POP			0x40
typedef struct FST_val
{
	BYTE	id;
	BYTE	amount;
	BYTE	type;
// GUI STUFF
	char	dest_stack;
	char	dest_unit;
	char	dest_slot;
	char	dest_id;
} *FST_valP;

typedef struct PAN_val
{
	BYTE	id;
	BYTE	panning;
// GUI STUFF
} *PAN_valP;

typedef struct OUT_val
{
	BYTE	id;
	BYTE	gain;
	BYTE	auxsend;
// GUI STUFF
} *OUT_valP;

#define ACC_OUT				0
#define	ACC_AUX				8
typedef struct ACC_val
{
	BYTE	id;
	BYTE	flags;
} *ACC_valP;

typedef struct FLD_val
{
	BYTE	id;
	BYTE	value;
// GUI STUFF
} *FLD_valP;

typedef struct GLITCH_val
{
	BYTE	id;
	BYTE	active;
	BYTE	dry;
	BYTE	dsize;
	BYTE	dpitch;
	BYTE	delay;
// GUI STUFF
	BYTE	guidelay;
} *GLITCH_valP;


/////////////////////////////////////////////////////////////////////////////////////
// workspace definitions
/////////////////////////////////////////////////////////////////////////////////////

typedef struct InstrumentWorkspace
{
	DWORD	release;
	DWORD	note;
	float	workspace[MAX_UNITS*MAX_UNIT_SLOTS];
	float	dlloutl;
	float	dlloutr;
	float	outl;
	float	outr;
} *InstrumentWorkspaceP;

typedef struct SynthObject
{
	DWORD Polyphony;
	char InstrumentNames[MAX_INSTRUMENTS][64];
	BYTE InstrumentValues[MAX_INSTRUMENTS][MAX_UNITS][MAX_UNIT_SLOTS];		// 16 instruments a 32 slots a 32 dowrds
	BYTE GlobalValues[MAX_UNITS][MAX_UNIT_SLOTS];								// 32 slots a 32 dwords
	InstrumentWorkspace InstrumentWork[MAX_INSTRUMENTS*MAX_POLYPHONY];
	InstrumentWorkspace GlobalWork;
	DWORD InstrumentSignalValid[MAX_INSTRUMENTS];
	DWORD GlobalSignalValid;
	float SignalTrace[MAX_INSTRUMENTS];
	int ControlInstrument[MAX_INSTRUMENTS];
	int VoiceIndex[MAX_INSTRUMENTS];
	int HighestSlotIndex[17];
} *SynthObjectP;

SynthObjectP Go4kVSTi_GetSynthObject();