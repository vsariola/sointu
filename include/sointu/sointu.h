#ifndef _SOINTU_H
#define _SOINTU_H

#pragma pack(push,1) // this should be fine for both Go and assembly
typedef struct Unit {
    float State[8];
    float Ports[8];
} Unit;

typedef struct Voice {
    int Note;
    int Release;
    float Inputs[8];
    float Reserved[6];
    struct Unit Units[63];
} Voice;

typedef struct DelayWorkspace {
    float Buffer[65536];
    float Dcin;
    float Dcout;
    float Filtstate;
} DelayWorkspace;

typedef struct SynthWorkspace {
    unsigned char Curvoices[32];
    float Left;
    float Right;
    float Aux[6];
    struct Voice Voices[32];
} SynthWorkspace;

typedef struct SampleOffset {
    unsigned int Start;
    unsigned short LoopStart;
    unsigned short LoopLength;
} SampleOffset;

typedef struct Synth {
    struct SynthWorkspace SynthWrk;
    struct DelayWorkspace DelayWrks[64]; // let's keep this as 64 for now, so the delays take 16 meg. If that's too little or too much, we can change this in future.    
    unsigned short DelayTimes[768];
    struct SampleOffset SampleOffsets[256];
    unsigned int RandSeed;
    unsigned int GlobalTick;    
    unsigned char Commands[32 * 64];
    unsigned char Values[32 * 64 * 8];
    unsigned int Polyphony;
    unsigned int NumVoices;
} Synth;
#pragma pack(pop)

#if UINTPTR_MAX == 0xffffffff // are we 32-bit?
#if defined(__clang__) || defined(__GNUC__)
#define CALLCONV __attribute__ ((stdcall))
#elif defined(_WIN32)
#define CALLCONV __stdcall // on 32-bit platforms, we just use stdcall, as all know it
#endif
#else // 64-bit
#define CALLCONV  // the asm will use honor honor correct x64 ABI on all 64-bit platforms
#endif

void CALLCONV su_load_gmdls(void);

// int su_render(Synth* synth, float* buffer, int* samples, int* time):
//      Renders samples until 'samples' number of samples are reached or 'time' number of
//      modulated time ticks are reached, whichever happens first. 'samples' and 'time' are
//      are passed by reference as the function modifies to tell how many samples were
//      actually rendered and how many time ticks were actually advanced.
//
// Parameters:
//      synth       pointer to the synthesizer used. RandSeed should be > 0 e.g. 1                  
//      buffer      audio sample buffer, L R L R ...
//      samples     pointer to the maximum number of samples to be rendered.
//                  buffer should have a length of 2 * maxsamples as the audio
//                  is stereo.
//      time        maximum modulated time rendered.
//
// The value referred by samples is changed to contain the actual number of samples rendered
// Similarly, the value referred by time is changed to contain the number of time ticks advanced.
// If samples_out == samples_in, then is must be that time_in <= time_out.
// If samples_out < samples_in, then time_out >= time_in. Note that it could happen that
// time_out > time_in, as it is modulated and the time could advance by 2 or more, so the loop
// exit condition would fire when the current time is already past time_in
//
// Returns an error code, which is actually just masked version of the FPU Status Word
// On a succesful run, the return value should be 0
// Error code bits:
//    bit  0        FPU invalid operation (stack over/underflow OR invalid arithmetic e.g. NaNs)
//    bit  2        Divide by zero occurred
//    bit  6        Stack overflow or underflow occurred
//    bits 11-13    The top pointer of the fpu stack. Any other value than 0 indicates that some values were left on the stack.
int CALLCONV su_render(Synth* synth, float* buffer, int* samples, int* time);

// Arithmetic opcode ids
extern const int su_add_id;
extern const int su_addp_id;
extern const int su_pop_id;
extern const int su_loadnote_id;
extern const int su_mul_id;
extern const int su_mulp_id;
extern const int su_push_id;
extern const int su_xch_id;

// Effect opcode ids
extern const int su_distort_id;
extern const int su_hold_id;
extern const int su_crush_id;
extern const int su_gain_id;
extern const int su_invgain_id;
extern const int su_filter_id;
extern const int su_clip_id;
extern const int su_pan_id;
extern const int su_delay_id;
extern const int su_compres_id;

// Flowcontrol opcode ids
extern const int su_advance_id;
extern const int su_speed_id;

// Sink opcode ids
extern const int su_out_id;
extern const int su_outaux_id;
extern const int su_aux_id;
extern const int su_send_id;

// Source opcode ids
extern const int su_envelope_id;
extern const int su_noise_id;
extern const int su_oscillat_id;
extern const int su_loadval_id;
extern const int su_receive_id;
extern const int su_in_id;

#endif // _SOINTU_H
