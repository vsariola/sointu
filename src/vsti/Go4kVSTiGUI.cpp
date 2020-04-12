#define _WIN32_WINNT	0x0400
#include <windows.h>
#include <winuser.h>
#include <commctrl.h>
#include "resource.h"
#include "Go4kVSTiGUI.h"
#include "Go4kVSTiCore.h"
#include <stdio.h>
#include <vector>
#include <map>
#include <string>
#include <math.h>

#define T_INSTRUMENT	0
#define T_GLOBAL		1
#define NUM_TABS		2

static HINSTANCE hInstance;
static UINT_PTR timer = 1;
static UINT_PTR backupTimer = 2;
static UINT_PTR timerID = 0;
static UINT_PTR backupTimerID = 0;
static HWND DialogWnd = NULL;
static HWND ModuleWnd[NUM_MODULES];
static HWND TabWnd[NUM_TABS];
static HWND SetWnd;
static HWND ScrollWnd;
static int SelectedIUnit = 0;
static int SelectedGUnit = 0;
static int SelectedTab = T_INSTRUMENT;
static int SelectedInstrument = 0;
static int LinkToInstrument[16] = { 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16 };
static int SetUnit = -1;
static char SliderValTxt[128];
static DWORD UnitCopyBuffer[MAX_UNIT_SLOTS];
static int InstrumentScrollPos[MAX_INSTRUMENTS];
static int GlobalScrollPos;
static int InitDone = false;

static SynthObjectP SynthObjP;

static char* UnitName[NUM_MODULES] =
{
	"-",
	"Envelope",
	"Oscillator",
	"Filter",
	"Distortion",
	"DelayLine",
	"Arithmetic",
	"Store",
	"Panning",
	"Output",
	"Accumulate",
	"Load",
	"Glitch"
};

// minimal signal precondition for each unit
static int UnitPreSignals[NUM_MODULES] = 
{
	0,	// none
	0,	// env
	0,	// vco
	1,	// vcf
	1,	// dst
	1,	// dll
	2,  // fop, signal count for arithmetic depends on mode
	1,	// fst
	1,	// pan
	2,	// out
	0,	// acc
	0,  // fld
	1,  // glitch
};

// signal post condition (relative to precondition)
static int UnitPostSignals[NUM_MODULES] = 
{
	0,	// none
	1,	// env
	1,	// vco
	0,	// vcf
	0,	// dst
	0,	// dll
	0,  // fop, signal count for arithmetic depends on mode
	0,	// fst
	1,	// pan
	-2,	// out
	2,	// acc
	1,	// fld
	0	// glitch
};

static char* UnitModulationTargetNames[][8] = 
{
	{ "", "", "", "", "", "", "", "" },
	{ "", "", "Gain", "Attack", "Decay", "", "Release", "" },
	{ "", "Transpose", "Detune", "Frequency", "Phase", "Color", "Shape", "Gain"},
	{ "", "", "", "", "Cutoff Frequency", "Resonance", "", "" },
	{ "", "", "Drive", "Sample&Hold Frequency", "", "", "", ""},
	{ "Pregain", "Feedback", "Dry", "Damp", "LFO Frequency", "LFO Depth", "", "" },
	{ "", "", "", "", "", "", "", "" },
	{ "", "", "", "", "", "", "", "" },
	{ "Panning", "", "", "", "", "", "", "" },
	{ "AUX", "Gain", "", "", "", "", "", "" },
	{ "", "", "", "", "", "", "", "" },
	{ "Value", "", "", "", "", "", "", "" },
	{ "Active", "Dry", "Delta Size", "Delta Pitch", "", "", "", "" },
};

static char* UnitModulationTargetShortNames[][8] = 
{
	{ "", "", "", "", "", "", "", "" },
	{ "", "", "Gain", "Attack", "Decay", "", "Release", "" },
	{ "", "Transp", "Detune", "Freq", "Phase", "Color", "Shape", "Gain"},
	{ "", "", "", "", "Cutoff", "Res", "", "" },
	{ "", "", "Drive", "Freq", "", "", "", ""},
	{ "Pregain", "Feedb", "Dry", "Damp", "LFO F", "LFO D", "", "" },
	{ "", "", "", "", "", "", "", "" },
	{ "", "", "", "", "", "", "", "" },
	{ "Pan", "", "", "", "", "", "", "" },
	{ "AUX", "Gain", "", "", "", "", "", "" },
	{ "", "", "", "", "", "", "", "" },
	{ "Value", "", "", "", "", "", "", "" },
	{ "Active", "Dry", "DSize", "DPitch", "", "", "", "" },
};

static char* delayName[33] = 
{	
	"1/32T",
	"1/32",
	"1/32D",
	"1/16T",
	"1/16",
	"1/16D",
	"1/8T",
	"1/8",
	"1/8D",
	"1/4T",
	"1/4",
	"1/4D",
	"1/2T",
	"1/2",
	"1/2D",
	"1T",
	"1",
	"1D",
	"2T",
	"2",
	"2D",
	"3/8",
	"5/8",
	"7/8",
	"9/8",
	"11/8",
	"13/8",
	"15/8",
	"3/4",
	"5/4",
	"7/4",
	"3/2",
	"3/2",
};

char* GetUnitString(BYTE* unit, char* unitname)
{
	char UnitDesc[128];
	UnitDesc[0]=0;
	
	if (unit[0] == M_VCO)
	{
		VCO_valP val = (VCO_valP)unit;
		char lfo[5];
		lfo[0] = 0;
		if (val->flags & VCO_LFO)
			sprintf(lfo, "-LFO");
		if (val->flags & VCO_SINE)
			sprintf(UnitDesc, "    (%s%s)","Sine",lfo);
		if (val->flags & VCO_TRISAW)
			sprintf(UnitDesc, "    (%s%s)","TriSaw",lfo);
		if (val->flags & VCO_PULSE)
			sprintf(UnitDesc, "    (%s%s)","Pulse",lfo);
		if (val->flags & VCO_NOISE)
			sprintf(UnitDesc, "    (%s%s)","Noise",lfo);
		if (val->flags & VCO_GATE)
			sprintf(UnitDesc, "    (%s%s)","Gate",lfo);
	}
	if (unit[0] == M_VCF)
	{
		VCF_valP val = (VCF_valP)unit;
		int ft = val->type & ~VCF_STEREO;
		if (ft == VCF_LOWPASS)
			sprintf(UnitDesc, "    (%s F:%d)","Low", val->freq);
		if (ft == VCF_HIGHPASS)
			sprintf(UnitDesc, "    (%s F:%d)","High", val->freq);
		if (ft == VCF_BANDPASS)
			sprintf(UnitDesc, "    (%s F:%d)","Band", val->freq);
		if (ft == VCF_BANDSTOP)
			sprintf(UnitDesc, "    (%s F:%d)","Notch", val->freq);
		if (ft == VCF_PEAK)
			sprintf(UnitDesc, "    (%s F:%d)","Peak", val->freq);
		if (ft == VCF_ALLPASS)
			sprintf(UnitDesc, "    (%s F:%d)","All", val->freq);		
	}
	if (unit[0] == M_FOP)
	{
		FOP_valP val = (FOP_valP)unit;
		if (val->flags == FOP_ADD)
			sprintf(UnitDesc, "    (%s)","+");
		if (val->flags == FOP_ADDP)
			sprintf(UnitDesc, "    (%s)","+/Pop");
		if (val->flags == FOP_MUL)
			sprintf(UnitDesc, "    (%s)","*");
		if (val->flags == FOP_MULP)
			sprintf(UnitDesc, "    (%s)","*/Pop");
		if (val->flags == FOP_POP)
			sprintf(UnitDesc, "    (%s)","Pop");
		if (val->flags == FOP_PUSH)
			sprintf(UnitDesc, "    (%s)","Push");
		if (val->flags == FOP_XCH)
			sprintf(UnitDesc, "    (%s)","Xch");
		if (val->flags == FOP_ADDP2)
			sprintf(UnitDesc, "    (%s)","2+/Pop");
		if (val->flags == FOP_LOADNOTE)
			sprintf(UnitDesc, "    (%s)","LoadNote");
		if (val->flags == FOP_MULP2)
			sprintf(UnitDesc, "    (%s)","2*/Pop");
	}
	if (unit[0] == M_DLL)
	{
		DLL_valP val = (DLL_valP)unit;
		if (val->reverb == 0)
			sprintf(UnitDesc, "    (%s)","Delay");
		else
			sprintf(UnitDesc, "    (%s)","Reverb");
	}
	if (unit[0] == M_FST)
	{
		FST_valP val = (FST_valP)unit;
		if (val->dest_unit == -1 || val->dest_slot == -1 || val->dest_id == -1)
		{
			sprintf(UnitDesc, "    (No Target)");
		}
		else
		{
			std::string st;
			if (val->type & FST_ADD)
				st = "+";
			if (val->type & FST_MUL)
				st = "*";
			if (val->type & FST_POP)
				st += ",Pop";

			if ((val->dest_stack != -1) && (val->dest_stack != SelectedInstrument))
			{
				if (val->dest_stack == MAX_INSTRUMENTS)
					sprintf(UnitDesc, "    (GU%d %s)%s", val->dest_unit+1, UnitModulationTargetShortNames[val->dest_id][val->dest_slot], st.c_str());
				else
					sprintf(UnitDesc, "    (%dU%d %s)%s",val->dest_stack+1, val->dest_unit+1, UnitModulationTargetShortNames[val->dest_id][val->dest_slot], st.c_str());
			}
			else
			{
				sprintf(UnitDesc, "    (U%d %s)%s", val->dest_unit+1, UnitModulationTargetShortNames[val->dest_id][val->dest_slot], st.c_str());
			}
		}
	}
	if (unit[0] == M_DST)
	{
		DST_valP val = (DST_valP)unit;
		sprintf(UnitDesc, "    (D:%d)", val->drive-64, val->snhfreq);
	}
	if (unit[0] == M_PAN)
	{
		PAN_valP val = (PAN_valP)unit;
		sprintf(UnitDesc, "    (%d)", val->panning-64);
	}
	if (unit[0] == M_ACC)
	{
		ACC_valP val = (ACC_valP)unit;
		if (val->flags == ACC_OUT)
			sprintf(UnitDesc, "    (%s)","Out");
		else
			sprintf(UnitDesc, "    (%s)","Aux");
	}
	if (unit[0] == M_OUT)
	{
		OUT_valP val = (OUT_valP)unit;
		sprintf(UnitDesc, "    (G:%d A:%d)",val->gain, val->auxsend);
	}
	if (unit[0] == M_FLD)
	{
		FLD_valP val = (FLD_valP)unit;
		sprintf(UnitDesc, "    (%d)", val->value-64);
	}
	if (unit[0] == M_GLITCH)
	{
		GLITCH_valP val = (GLITCH_valP)unit;
		sprintf(UnitDesc, "    (%s)","Glitch");
	}

	sprintf(unitname, "%s%s", UnitName[unit[0]], UnitDesc);
	return unitname;
}

void UpdateDelayTimes(DLL_valP unit)
{	
	// if reverb skip
	if (unit->reverb)
		return;

	Go4kVSTi_UpdateDelayTimes();

	int delay;
	char text[10];
	DLL_valP v = (DLL_valP)unit;
	if (v->synctype == 2)
	{
	}
	else
	{
		if (v->synctype == 1)
		{
			sprintf(text, "%s", delayName[v->guidelay>>2]);
		}
		else
		{
			delay = v->guidelay*16;
			sprintf(text, "%.2f ms", (float)delay/44.100f);
		}
		sprintf(SliderValTxt, "%s", text);
		SetWindowText(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), SliderValTxt);
	}
}

void UpdateDelayTimes(GLITCH_valP unit)
{	
	Go4kVSTi_UpdateDelayTimes();

	int delay;
	char text[10];
	GLITCH_valP v = (GLITCH_valP)unit;
	sprintf(text, "%s", delayName[v->guidelay>>2]);
	sprintf(SliderValTxt, "%s", text);
	SetWindowText(GetDlgItem(ModuleWnd[M_GLITCH], IDC_GLITCH_DTIME_VAL), SliderValTxt);
}

// CB for the main DLG
BOOL CALLBACK MainDialogProc(HWND hwndDlg, UINT message, WPARAM wParam, LPARAM lParam) 
{ 
	switch (message) 
    { 
		case WM_INITDIALOG:
		{			
			HWND tab = GetDlgItem(hwndDlg, IDC_MAINTAB);
			TCITEM item;
			item.mask = TCIF_TEXT;
			item.pszText = "Instrument";
			SendDlgItemMessage(hwndDlg, IDC_MAINTAB, TCM_INSERTITEM, (WPARAM)0, (LPARAM)(&item));
			item.pszText = "Global";
			SendDlgItemMessage(hwndDlg, IDC_MAINTAB, TCM_INSERTITEM, (WPARAM)2, (LPARAM)(&item));

			const char* instrument[] = {"1", "2", "3", "4", "5", "6", "7", "8",
				"9", "10", "11", "12", "13", "14", "15", "16" };
			for (int i = 0; i < 16; i++)
			{
				SendDlgItemMessage(hwndDlg, IDC_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)(instrument[i]));
				SendDlgItemMessage(hwndDlg, IDC_INSTRUMENTLINK, CB_ADDSTRING, (WPARAM)0, (LPARAM)(instrument[i]));
			}
			SendDlgItemMessage(hwndDlg, IDC_INSTRUMENTLINK, CB_ADDSTRING, (WPARAM)0, (LPARAM)("None"));
			SendDlgItemMessage(hwndDlg, IDC_INSTRUMENT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);
			SendDlgItemMessage(hwndDlg, IDC_INSTRUMENT, CB_SETCURSEL, (WPARAM)16, (LPARAM)0);

			const char* patternsize[] = {"8", "16", "32", "64"};
			for (int i = 0; i < 4; i++)
				SendDlgItemMessage(hwndDlg, IDC_PATTERN_SIZE, CB_ADDSTRING, (WPARAM)0, (LPARAM)(patternsize[i]));
			SendDlgItemMessage(hwndDlg, IDC_PATTERN_SIZE, CB_SETCURSEL, (WPARAM)1, (LPARAM)0);

			const char* patternquant[] = {"1/4", "1/8", "1/16", "1/32", "1/64", "1/128"};
			for (int i = 0; i < 6; i++)
				SendDlgItemMessage(hwndDlg, IDC_PATTERN_QUANT, CB_ADDSTRING, (WPARAM)0, (LPARAM)(patternquant[i]));
			SendDlgItemMessage(hwndDlg, IDC_PATTERN_QUANT, CB_SETCURSEL, (WPARAM)2, (LPARAM)0);

			const char* objformat[] = {"Windows OBJ", "Linux ELF", "OSX MACHO"};
			for (int i = 0; i < 3; i++)
				SendDlgItemMessage(hwndDlg, IDC_OBJFORMAT, CB_ADDSTRING, (WPARAM)0, (LPARAM)(objformat[i]));
			SendDlgItemMessage(hwndDlg, IDC_OBJFORMAT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);

			const char* polyphony[] = { "1xPolyphone", "2xPolyphone" };
			for (int i = 0; i < MAX_POLYPHONY; i++)
				SendDlgItemMessage(hwndDlg, IDC_POLYPHONY, CB_ADDSTRING, (WPARAM)0, (LPARAM)(polyphony[i]));
			SendDlgItemMessage(hwndDlg, IDC_POLYPHONY, CB_SETCURSEL, (WPARAM)1, (LPARAM)0);

			SendDlgItemMessage(hwndDlg, IDC_CLIPOUTPUT, BM_SETCHECK, 1, 0);
			SendDlgItemMessage(hwndDlg, IDC_RECORDBUSYSIGNAL, BM_SETCHECK, 1, 0);

			return TRUE;
		}
		case WM_COMMAND: 
		{
			// channel combo box 
			if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_INSTRUMENT))
			{
				if (HIWORD(wParam) == CBN_SELCHANGE)
				{
					SelectedInstrument = SendDlgItemMessage(hwndDlg, IDC_INSTRUMENT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
					if (SendDlgItemMessage(hwndDlg, IDC_SOLO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
						Go4kVSTi_Solo(SelectedInstrument, true);
					else
						Go4kVSTi_Solo(SelectedInstrument, false);
					// get values from synth and set gui accordingly ...
					UpdateControls(SelectedInstrument);
					// reset scroll pos
					int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
					SCROLLINFO sinfo;
					sinfo.cbSize = sizeof(SCROLLINFO);
					sinfo.fMask = SIF_POS;
					sinfo.nPos = InstrumentScrollPos[SelectedInstrument];
					SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
					ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
				}
				return TRUE;
			}
			// link to combo box 
			if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_INSTRUMENTLINK))
			{
				if (HIWORD(wParam) == CBN_SELCHANGE)
				{
					int linkToInstrument = SendDlgItemMessage(hwndDlg, IDC_INSTRUMENTLINK, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
					if (SelectedInstrument == linkToInstrument)
					{
						MessageBox(DialogWnd, "Instrument cannot be linked to itself!", "Info", MB_OK);
						return TRUE;
					}
					if (linkToInstrument < 16)
					{
						// compare instruments
						for (int u = 0; u < MAX_UNITS; u++)
						{
							// special case, compare manually
							if (SynthObjP->InstrumentValues[SelectedInstrument][u][0] == M_DLL)
							{
								DLL_valP ds = (DLL_valP)SynthObjP->InstrumentValues[SelectedInstrument][u];
								DLL_valP dl = (DLL_valP)SynthObjP->InstrumentValues[linkToInstrument][u];
								if (ds->pregain != dl->pregain ||
									ds->dry != dl->dry ||
									ds->feedback != dl->feedback ||
									ds->damp != dl->damp ||
									ds->freq != dl->freq ||
									ds->depth != dl->depth ||
									ds->guidelay != dl->guidelay ||
									ds->synctype != dl->synctype ||
									ds->leftreverb != dl->leftreverb ||
									ds->reverb != dl->reverb)
								{
									MessageBox(DialogWnd, "Instruments cannot be linked as they differ!", "Info", MB_OK);
									SendDlgItemMessage(hwndDlg, IDC_INSTRUMENTLINK, CB_SETCURSEL, (WPARAM)16, (LPARAM)0);
									return TRUE;
								}
							}
							else if (SynthObjP->InstrumentValues[SelectedInstrument][u][0] == M_GLITCH)
							{
								GLITCH_valP ds = (GLITCH_valP)SynthObjP->InstrumentValues[SelectedInstrument][u];
								GLITCH_valP dl = (GLITCH_valP)SynthObjP->InstrumentValues[linkToInstrument][u];
								if (ds->active != dl->active ||
									ds->dry != dl->dry ||
									ds->dsize != dl->dsize ||
									ds->dpitch != dl->dpitch ||
									ds->guidelay != dl->guidelay
									)
								{
									MessageBox(DialogWnd, "Instruments cannot be linked as they differ!", "Info", MB_OK);
									SendDlgItemMessage(hwndDlg, IDC_INSTRUMENTLINK, CB_SETCURSEL, (WPARAM)16, (LPARAM)0);
									return TRUE;
								}
							}
							else
							{
								if (memcmp(SynthObjP->InstrumentValues[SelectedInstrument][u], SynthObjP->InstrumentValues[linkToInstrument][u], MAX_UNIT_SLOTS))
								{
									MessageBox(DialogWnd, "Instruments cannot be linked as they differ!", "Info", MB_OK);
									SendDlgItemMessage(hwndDlg, IDC_INSTRUMENTLINK, CB_SETCURSEL, (WPARAM)16, (LPARAM)0);
									return TRUE;
								}
							}
						}
					}
					// set link
					LinkToInstrument[SelectedInstrument] = linkToInstrument;
				}
				return TRUE;
			}
			// Polyphony
			else if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_POLYPHONY))
			{
				if (HIWORD(wParam) == CBN_SELCHANGE)
				{
					SynthObjP->Polyphony = 1+SendDlgItemMessage(hwndDlg, IDC_POLYPHONY, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
					// get values from synth and set gui accordingly ...
					UpdateControls(SelectedInstrument);
					Go4kVSTi_UpdateDelayTimes();
					Go4kVSTi_ClearDelayLines();
				}
				return TRUE;
			}
			// solo check box
			else if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_SOLO))
			{
				if (SendDlgItemMessage(hwndDlg, IDC_SOLO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
					Go4kVSTi_Solo(SelectedInstrument, true);
				else
					Go4kVSTi_Solo(SelectedInstrument, false);
				return TRUE;
			}
			else if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_INSTRUMENT_NAME))
			{
				if (HIWORD(wParam) == EN_CHANGE)
				{
					GetDlgItemText(hwndDlg, IDC_INSTRUMENT_NAME, (LPSTR)&(SynthObjP->InstrumentNames[SelectedInstrument]), 127);
					int stsel = SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
					SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
					SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"Local");
					for (int i = 0; i < MAX_INSTRUMENTS; i++)
					{
						char instname[128];
						sprintf(instname, "%d: %s", i+1, SynthObjP->InstrumentNames[i]);
						SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)instname);
					}
					SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"Global");
					SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_SETCURSEL, (WPARAM)stsel, (LPARAM)0);
					return TRUE;
				}
			}
			// always on top check box
			else if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_ALWAYSONTOP))
			{
				if (SendDlgItemMessage(hwndDlg, IDC_ALWAYSONTOP, BM_GETCHECK, 0, 0)==BST_CHECKED)
				{
					SetWindowPos(hwndDlg, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOMOVE | SWP_NOSIZE | SWP_FRAMECHANGED);					
				}
				else
				{
					SetWindowPos(hwndDlg, HWND_NOTOPMOST, 0, 0, 0, 0, SWP_NOMOVE | SWP_NOSIZE | SWP_FRAMECHANGED);
				}
				return TRUE;
			}
			else
			{
				// global Buttons
				switch (LOWORD(wParam)) 
			    { 
					case IDC_LOAD_PATCH:
					{
						char lpstrFilter[256] =
						{'4','k','l','a','n','g',' ','P','a','t','c','h', 0 ,
						'*','.','4','k','p', 0, 
						0};
						char lpstrFile[4096] = "*.4kp";
						char lpstrDirectory[4096];
						GetCurrentDirectory(4096, lpstrDirectory);

						OPENFILENAME ofn =
						{
							sizeof(OPENFILENAME),
							DialogWnd,
							GetModuleHandle(NULL),
							lpstrFilter,
							NULL,
							0,
							1,
							lpstrFile,
							256,
							NULL,
							0,
							lpstrDirectory,
							NULL,
							0,
							0,
							0,
							"4kp",
							0,
							NULL,
							NULL		
						};
						
						if (GetOpenFileName(&ofn))
						{
							// reset scroll pos
							int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
							SCROLLINFO sinfo;
							sinfo.cbSize = sizeof(SCROLLINFO);
							sinfo.fMask = SIF_POS;
							sinfo.nPos = 0;
							SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
							ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
							for (int i=0; i<MAX_INSTRUMENTS; i++)
								InstrumentScrollPos[i] = 0;
							GlobalScrollPos = 0;
							
							for (int i = 0; i < MAX_INSTRUMENTS; i++)
							{
								SynthObjP->InstrumentSignalValid[i] = 0;
							}
							SynthObjP->GlobalSignalValid = 0;
							Go4kVSTi_LoadPatch(ofn.lpstrFile);
							//instrument check
							for (int i = 0; i < MAX_INSTRUMENTS; i++)
							{
								UpdateSignalCount(i);							
								LinkToInstrument[i] = 16;
								// try setting up instrument links
								if (i > 0)
								{
									for (int j = 0; j < i; j++)
									{
										int linkToInstrument = j;
										// compare instruments
										for (int u = 0; u < MAX_UNITS; u++)
										{
											// special case, compare manually
											if (SynthObjP->InstrumentValues[i][u][0] == M_DLL)
											{
												DLL_valP ds = (DLL_valP)SynthObjP->InstrumentValues[i][u];
												DLL_valP dl = (DLL_valP)SynthObjP->InstrumentValues[j][u];
												if (ds->pregain != dl->pregain ||
													ds->dry != dl->dry ||
													ds->feedback != dl->feedback ||
													ds->damp != dl->damp ||
													ds->freq != dl->freq ||
													ds->depth != dl->depth ||
													ds->guidelay != dl->guidelay ||
													ds->synctype != dl->synctype ||
													ds->leftreverb != dl->leftreverb ||
													ds->reverb != dl->reverb)
												{
													linkToInstrument = 16;
													break;
												}
											}
											else if (SynthObjP->InstrumentValues[i][u][0] == M_GLITCH)
											{
												GLITCH_valP ds = (GLITCH_valP)SynthObjP->InstrumentValues[i][u];
												GLITCH_valP dl = (GLITCH_valP)SynthObjP->InstrumentValues[j][u];
												if (ds->active != dl->active ||
													ds->dry != dl->dry ||
													ds->dsize != dl->dsize ||
													ds->dpitch != dl->dpitch ||
													ds->guidelay != dl->guidelay
													)
												{
													linkToInstrument = 16;
													break;
												}
											}
											else
											{
												if (memcmp(SynthObjP->InstrumentValues[i][u], SynthObjP->InstrumentValues[j][u], MAX_UNIT_SLOTS))
												{
													linkToInstrument = 16;
													break;
												}
											}										
										}
										// set link
										if (linkToInstrument != 16)
										{
											LinkToInstrument[i] = linkToInstrument;
											break;
										}
									}
								}
							}
							UpdateControls(SelectedInstrument);
						}
						return TRUE;
					}
					case IDC_SAVE_PATCH:
					{
						char lpstrFilter[256] =
						{'4','k','l','a','n','g',' ','P','a','t','c','h', 0 ,
						'*','.','4','k','p', 0, 
						0};
						char lpstrFile[4096] = "*.4kp";
						char lpstrDirectory[4096];
						GetCurrentDirectory(4096, lpstrDirectory);

						OPENFILENAME ofn =
						{
							sizeof(OPENFILENAME),
							DialogWnd,
							GetModuleHandle(NULL),
							lpstrFilter,
							NULL,
							0,
							1,
							lpstrFile,
							256,
							NULL,
							0,
							lpstrDirectory,
							NULL,
							OFN_OVERWRITEPROMPT,
							0,
							0,
							"4kp",
							0,
							NULL,
							NULL		
						};

						if (GetSaveFileName(&ofn))
						{
							Go4kVSTi_SavePatch(ofn.lpstrFile);
						}
						return TRUE;
					}
					case IDC_RESET_PATCH:
					{
						if (MessageBox(0, "Do you really want to reset the patch?", "Info", MB_YESNO) == IDYES)
						{
							// reset scroll pos
							int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
							SCROLLINFO sinfo;
							sinfo.cbSize = sizeof(SCROLLINFO);
							sinfo.fMask = SIF_POS;
							sinfo.nPos = 0;
							SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
							ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
							for (int i=0; i<MAX_INSTRUMENTS; i++)
								InstrumentScrollPos[i] = 0;
							GlobalScrollPos = 0;

							for (int i = 0; i < MAX_INSTRUMENTS; i++)
							{
								SynthObjP->InstrumentSignalValid[i] = 0;
							}
							SynthObjP->GlobalSignalValid = 0;
							Go4kVSTi_ResetPatch();
							UpdateControls(SelectedInstrument);						
						}
						return TRUE;
					}
					case IDC_LOAD_INSTRUMENT:
					{
						char lpstrFilter[256] =
						{'4','k','l','a','n','g',' ','I','n','s','t','r','u','m','e','n','t', 0 ,
						'*','.','4','k','i', 0, 
						0};
						char lpstrFile[4096] = "*.4ki";
						char lpstrDirectory[4096];
						GetCurrentDirectory(4096, lpstrDirectory);

						OPENFILENAME ofn =
						{
							sizeof(OPENFILENAME),
							DialogWnd,
							GetModuleHandle(NULL),
							lpstrFilter,
							NULL,
							0,
							1,
							lpstrFile,
							256,
							NULL,
							0,
							lpstrDirectory,
							NULL,
							0,
							0,
							0,
							"4ki",
							0,
							NULL,
							NULL		
						};
						
						if (GetOpenFileName(&ofn))
						{
							// reset scroll pos
							int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
							SCROLLINFO sinfo;
							sinfo.cbSize = sizeof(SCROLLINFO);
							sinfo.fMask = SIF_POS;
							sinfo.nPos = 0;
							SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
							ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
														
							UpdateControls(SelectedInstrument);
							if (SelectedTab == T_INSTRUMENT)
							{
								InstrumentScrollPos[SelectedInstrument] = 0;
								SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
								Go4kVSTi_LoadInstrument(ofn.lpstrFile, (char)SelectedInstrument);
								LinkToInstrument[SelectedInstrument] = 16;
							}
							else
							{
								GlobalScrollPos = 0;
								SynthObjP->GlobalSignalValid = 0;
								Go4kVSTi_LoadInstrument(ofn.lpstrFile, (char)16);
								LinkToInstrument[SelectedInstrument] = 16;
							}
							UpdateControls(SelectedInstrument);			
						}
						return TRUE;
					}
					case IDC_SAVE_INSTRUMENT:
					{
						char lpstrFilter[256] =
						{'4','k','l','a','n','g',' ','I','n','s','t','r','u','m','e','n','t', 0 ,
						'*','.','4','k','i', 0, 
						0};
						char lpstrFile[4096] = "*.4ki";
						char lpstrDirectory[4096];
						GetCurrentDirectory(4096, lpstrDirectory);

						OPENFILENAME ofn =
						{
							sizeof(OPENFILENAME),
							DialogWnd,
							GetModuleHandle(NULL),
							lpstrFilter,
							NULL,
							0,
							1,
							lpstrFile,
							256,
							NULL,
							0,
							lpstrDirectory,
							NULL,
							OFN_OVERWRITEPROMPT,
							0,
							0,
							"4ki",
							0,
							NULL,
							NULL		
						};

						if (GetSaveFileName(&ofn))
						{
							if (SelectedTab == T_INSTRUMENT)
								Go4kVSTi_SaveInstrument(ofn.lpstrFile, (char)SelectedInstrument);
							else
								Go4kVSTi_SaveInstrument(ofn.lpstrFile, (char)16);
						}
						return TRUE;
					}
					case IDC_RESET_INSTRUMENT:
					{
						if (MessageBox(0, "Do you really want to reset the instrument?", "Info", MB_YESNO) == IDYES)
						{
							if (SelectedTab == T_INSTRUMENT)
							{
								// reset scroll pos
								int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
								SCROLLINFO sinfo;
								sinfo.cbSize = sizeof(SCROLLINFO);
								sinfo.fMask = SIF_POS;
								sinfo.nPos = 0;
								SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
								ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
								InstrumentScrollPos[SelectedInstrument] = 0;
								
								SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
								Go4kVSTi_ResetInstrument(SelectedInstrument);
							}
							else
							{
								// reset scroll pos
								int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
								SCROLLINFO sinfo;
								sinfo.cbSize = sizeof(SCROLLINFO);
								sinfo.fMask = SIF_POS;
								sinfo.nPos = 0;
								SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
								ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
								GlobalScrollPos = 0;

								SynthObjP->GlobalSignalValid = 0;
								Go4kVSTi_ResetGlobal();
							}
							UpdateControls(SelectedInstrument);						
						}
						return TRUE;
					}
					case IDC_UNIT_LOAD:
					{
						char lpstrFilter[256] =
						{'4','k','l','a','n','g',' ','U','n','i','t', 0 ,
						'*','.','4','k','u', 0, 
						0};
						char lpstrFile[4096] = "*.4ku";
						char lpstrDirectory[4096];
						GetCurrentDirectory(4096, lpstrDirectory);

						OPENFILENAME ofn =
						{
							sizeof(OPENFILENAME),
							DialogWnd,
							GetModuleHandle(NULL),
							lpstrFilter,
							NULL,
							0,
							1,
							lpstrFile,
							256,
							NULL,
							0,
							lpstrDirectory,
							NULL,
							0,
							0,
							0,
							"4ku",
							0,
							NULL,
							NULL		
						};
						
						if (GetOpenFileName(&ofn))
						{
							BYTE* uslot;
							if (SelectedTab == T_INSTRUMENT)
							{
								SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
								uslot = SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit];
							}
							else
							{
								SynthObjP->GlobalSignalValid = 0;
								uslot = SynthObjP->GlobalValues[SelectedGUnit];
							}
							Go4kVSTi_LoadUnit(ofn.lpstrFile, uslot);
							UpdateControls(SelectedInstrument);
						}
						return TRUE;
					}
					case IDC_UNIT_SAVE:
					{
						char lpstrFilter[256] =
						{'4','k','l','a','n','g',' ','U','n','i','t', 0 ,
						'*','.','4','k','u', 0, 
						0};
						char lpstrFile[4096] = "*.4ku";
						char lpstrDirectory[4096];
						GetCurrentDirectory(4096, lpstrDirectory);

						OPENFILENAME ofn =
						{
							sizeof(OPENFILENAME),
							DialogWnd,
							GetModuleHandle(NULL),
							lpstrFilter,
							NULL,
							0,
							1,
							lpstrFile,
							256,
							NULL,
							0,
							lpstrDirectory,
							NULL,
							OFN_OVERWRITEPROMPT,
							0,
							0,
							"4ku",
							0,
							NULL,
							NULL		
						};

						if (GetSaveFileName(&ofn))
						{
							BYTE* uslot;
							if (SelectedTab == T_INSTRUMENT)
							{
								uslot = SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit];
							}
							else
							{
								uslot = SynthObjP->GlobalValues[SelectedGUnit];
							}
							Go4kVSTi_SaveUnit(ofn.lpstrFile, uslot);
						}
						return TRUE;
					}
					case IDC_RECORD_BUTTON:
					{
						EnableWindow(GetDlgItem(DialogWnd, IDC_RECORD_BUTTON), false);
						EnableWindow(GetDlgItem(DialogWnd, IDC_STOP_BUTTON), true);
						int patternsize = 16;
						int psize = SendDlgItemMessage(hwndDlg, IDC_PATTERN_SIZE, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
						if (psize == 0)
							patternsize = 8;
						if (psize == 2)
							patternsize = 32;
						if (psize == 3)
							patternsize = 64;
						float patternquant = 1.0;
						int pquant = SendDlgItemMessage(hwndDlg, IDC_PATTERN_QUANT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
						if (pquant == 0)
							patternquant = 0.25;
						if (pquant == 1)
							patternquant = 0.5;
						if (pquant == 3)
							patternquant = 2.0;
						if (pquant == 4)
							patternquant = 4.0;
						if (pquant == 5)
							patternquant = 8.0;

						bool recordingNoise = SendDlgItemMessage(hwndDlg, IDC_RECORDBUSYSIGNAL, BM_GETCHECK, 0, 0) == BST_CHECKED;

						Go4kVSTi_Record(true, recordingNoise, patternsize, patternquant);
						return TRUE;
					}
					case IDC_STOP_BUTTON:
					{
						EnableWindow(GetDlgItem(DialogWnd, IDC_RECORD_BUTTON), true);
						EnableWindow(GetDlgItem(DialogWnd, IDC_STOP_BUTTON), false);
						Go4kVSTi_Record(false, false, 0, 0);
						return TRUE;
					}
					case IDC_PANIC:
					{
						Go4kVSTi_Panic();
						return TRUE;
					}
					case IDC_UNIT_RESET:
					{
						if (SelectedTab == T_INSTRUMENT)
							Go4kVSTi_InitInstrumentSlot(SelectedInstrument, SelectedIUnit, SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit][0]);
						else
							Go4kVSTi_InitGlobalSlot(SelectedGUnit, SynthObjP->GlobalValues[SelectedGUnit][0]);
						UpdateControls(SelectedInstrument);	
						return TRUE;
					}
					case IDC_UNIT_COPY:
					{
						if (SelectedTab == T_INSTRUMENT)
							memcpy(UnitCopyBuffer, SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit], MAX_UNIT_SLOTS);
						else
							memcpy(UnitCopyBuffer, SynthObjP->GlobalValues[SelectedGUnit], MAX_UNIT_SLOTS);
						return TRUE;
					}
					case IDC_UNIT_PASTE:
					{
						if (SelectedTab == T_INSTRUMENT)
							memcpy(SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit], UnitCopyBuffer, MAX_UNIT_SLOTS);
						else
							memcpy(SynthObjP->GlobalValues[SelectedGUnit], UnitCopyBuffer, MAX_UNIT_SLOTS);
						UpdateControls(SelectedInstrument);	
						return TRUE;
					}
					return FALSE;
				}
				return FALSE;
			}
			return FALSE;
		}
/*		case WM_MOUSEWHEEL:
		{
			if (InitDone)
			{
				int scrollw = 16*((int)((short)HIWORD(wParam)))/120;

				int oldpos = GetScrollPos(ScrollWnd, SB_VERT);				
				SetScrollPos(ScrollWnd, SB_VERT, oldpos-scrollw, true);
				int newpos = GetScrollPos(ScrollWnd, SB_VERT);
				ScrollWindow(ScrollWnd, 0, oldpos - newpos, NULL, NULL);		
				if (SelectedTab == T_INSTRUMENT)
				{
					InstrumentScrollPos[SelectedInstrument] = newpos;
				}
				else if (SelectedTab == T_GLOBAL)
				{
					GlobalScrollPos = newpos;
				}
				return TRUE;
			}
			break;
		}
*/
		case WM_NOTIFY:
		{
			if (((LPNMHDR)lParam)->hwndFrom == GetDlgItem(hwndDlg, IDC_MAINTAB))
			{
				if (((LPNMHDR)lParam)->code == TCN_SELCHANGE)
				{
					int i = TabCtrl_GetCurSel(GetDlgItem(hwndDlg, IDC_MAINTAB));
					TabChanged(i);
					return TRUE;
				}					
			}
			break;
		}
		case WM_CLOSE:
		{
			ShowWindow( DialogWnd, SW_SHOWMINIMIZED );
			return TRUE;
		}
		case WM_DROPFILES:
		{
			char DropFileName[4096];
			DragQueryFile((HDROP)wParam, 0, DropFileName, 4096);
			std::string filename = DropFileName;
			int slen = filename.length();
			bool invalidFile = false;
			if (slen > 4)
			{
				std::string extension = filename.substr(slen-4, 4);
				// dropped patch?
				if (!extension.compare(".4kp"))
				{
					// reset scroll pos
					int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
					SCROLLINFO sinfo;
					sinfo.cbSize = sizeof(SCROLLINFO);
					sinfo.fMask = SIF_POS;
					sinfo.nPos = 0;
					SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
					ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
					for (int i=0; i<MAX_INSTRUMENTS; i++)
						InstrumentScrollPos[i] = 0;
					GlobalScrollPos = 0;

					for (int i = 0; i < MAX_INSTRUMENTS; i++)
					{
						SynthObjP->InstrumentSignalValid[i] = 0;
					}
					SynthObjP->GlobalSignalValid = 0;
					Go4kVSTi_LoadPatch(DropFileName);
					//instrument check
					for (int i = 0; i < MAX_INSTRUMENTS; i++)
					{
						UpdateSignalCount(i);
						LinkToInstrument[i] = 16;
						// try setting up instrument links
						if (i > 0)
						{
							for (int j = 0; j < i; j++)
							{
								int linkToInstrument = j;
								// compare instruments
								for (int u = 0; u < MAX_UNITS; u++)
								{
									// special case, compare manually
									if (SynthObjP->InstrumentValues[i][u][0] == M_DLL)
									{
										DLL_valP ds = (DLL_valP)SynthObjP->InstrumentValues[i][u];
										DLL_valP dl = (DLL_valP)SynthObjP->InstrumentValues[j][u];
										if (ds->pregain != dl->pregain ||
											ds->dry != dl->dry ||
											ds->feedback != dl->feedback ||
											ds->damp != dl->damp ||
											ds->freq != dl->freq ||
											ds->depth != dl->depth ||
											ds->guidelay != dl->guidelay ||
											ds->synctype != dl->synctype ||
											ds->leftreverb != dl->leftreverb ||
											ds->reverb != dl->reverb)
										{
											linkToInstrument = 16;
											break;
										}										
									}
									else if (SynthObjP->InstrumentValues[i][u][0] == M_GLITCH)
									{
										GLITCH_valP ds = (GLITCH_valP)SynthObjP->InstrumentValues[i][u];
										GLITCH_valP dl = (GLITCH_valP)SynthObjP->InstrumentValues[j][u];
										if (ds->active != dl->active ||
											ds->dry != dl->dry ||
											ds->dsize != dl->dsize ||
											ds->dpitch != dl->dpitch ||
											ds->guidelay != dl->guidelay
											)
										{
											linkToInstrument = 16;
											break;
										}
									}
									else
									{
										if (memcmp(SynthObjP->InstrumentValues[i][u], SynthObjP->InstrumentValues[j][u], MAX_UNIT_SLOTS))
										{
											linkToInstrument = 16;
											break;
										}
									}										
								}
								// set link
								if (linkToInstrument != 16)
								{
									LinkToInstrument[i] = linkToInstrument;
									break;
								}
							}
						}
					}
					UpdateControls(SelectedInstrument);
				}
				// dropped instrument?
				else if (!extension.compare(".4ki"))
				{
					// reset scroll pos
					int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
					SCROLLINFO sinfo;
					sinfo.cbSize = sizeof(SCROLLINFO);
					sinfo.fMask = SIF_POS;
					sinfo.nPos = 0;
					SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
					ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
												
					UpdateControls(SelectedInstrument);
					if (SelectedTab == T_INSTRUMENT)
					{
						InstrumentScrollPos[SelectedInstrument] = 0;
						SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
						Go4kVSTi_LoadInstrument(DropFileName, (char)SelectedInstrument);
						LinkToInstrument[SelectedInstrument] = 16;
					}
					else
					{
						GlobalScrollPos = 0;
						SynthObjP->GlobalSignalValid = 0;
						Go4kVSTi_LoadInstrument(DropFileName, (char)16);
					}
					UpdateControls(SelectedInstrument);
				}
				// dropped shit!
				else
				{
					invalidFile = true;
				}
			}
			else
			{
				invalidFile = true;
			}
			if (invalidFile)
			{
				MessageBox(0,DropFileName,"Only patches(4kp) and instruments(4ki) allowed!",MB_OK);
			}
			return TRUE;
		}
	} 
    return FALSE; 
} 

// CB for the tabs
BOOL CALLBACK ModuleSettingsDialogProc(HWND hwndDlg, UINT message, WPARAM wParam, LPARAM lParam) 
{ 
	switch (message) 
    { 
        case WM_COMMAND: 
			return ButtonPressed(wParam, lParam);

		case WM_HSCROLL:
			return ScrollbarChanged(hwndDlg, wParam, lParam);		
    } 
    return FALSE; 
} 

// CB for the tabs
BOOL CALLBACK ScrollDialogProc(HWND hwndDlg, UINT message, WPARAM wParam, LPARAM lParam) 
{ 
	switch (message) 
    {
		case WM_VSCROLL:
		{
			RECT rect;
			GetWindowRect(ScrollWnd, &rect);
			int pagesize = 1+rect.bottom - rect.top;
			int linesize = 16;
			int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
			switch (LOWORD(wParam))
			{	
				
				case SB_BOTTOM: // Scrolls to the lower right. 
					break;
				case SB_ENDSCROLL: // Ends scroll. 
					break;
				case SB_LINEDOWN: // Scrolls one line down. 
					SetScrollPos(ScrollWnd, SB_VERT, oldpos+linesize, true);
					break;
				case SB_LINEUP: // Scrolls one line up. 
					SetScrollPos(ScrollWnd, SB_VERT, oldpos-linesize, true);
					break;
				case SB_PAGEDOWN: // Scrolls one page down. 
					SetScrollPos(ScrollWnd, SB_VERT, oldpos+pagesize, true);
					break;
				case SB_PAGEUP: // Scrolls one page up. 
					SetScrollPos(ScrollWnd, SB_VERT, oldpos-pagesize, true);
					break;
				case SB_THUMBPOSITION: // The user has dragged the scroll box (thumb) and released the mouse button. The high-order word indicates the position of the scroll box at the end of the drag operation. 
				case SB_THUMBTRACK: // The user is dragging the scroll box. This message is sent repeatedly until the user releases the mouse button. The high-order word indicates the position that the scroll box has been dragged to. 
					SetScrollPos(ScrollWnd, SB_VERT, HIWORD(wParam), true);
					break;
				case SB_TOP: // Scrolls to the upper left.
					break;
			}
			int newpos = GetScrollPos(ScrollWnd, SB_VERT);
			ScrollWindow(ScrollWnd, 0, oldpos - newpos, NULL, NULL);		

			if (SelectedTab == T_INSTRUMENT)
			{
				InstrumentScrollPos[SelectedInstrument] = newpos;
			}
			else if (SelectedTab == T_GLOBAL)
			{
				GlobalScrollPos = newpos;
			}
			return TRUE;
		}
/*		case WM_MOUSEWHEEL:
		{
			if (InitDone)
			{
				int scrollw = 16*((int)((short)HIWORD(wParam)))/120;

				int oldpos = GetScrollPos(ScrollWnd, SB_VERT);				
				SetScrollPos(ScrollWnd, SB_VERT, oldpos-scrollw, true);
				int newpos = GetScrollPos(ScrollWnd, SB_VERT);
				ScrollWindow(ScrollWnd, 0, oldpos - newpos, NULL, NULL);		
				if (SelectedTab == T_INSTRUMENT)
				{
					InstrumentScrollPos[SelectedInstrument] = newpos;
				}
				else if (SelectedTab == T_GLOBAL)
				{
					GlobalScrollPos = newpos;
				}
				return TRUE;
			}
			break;
		}
*/
		default:
			break;
	} 
    return FALSE; 
}

// CB for the tabs
BOOL CALLBACK StackDialogProc(HWND hwndDlg, UINT message, WPARAM wParam, LPARAM lParam) 
{ 
	switch (message) 
    { 
        case WM_COMMAND: 
			return StackButtonPressed(wParam);
    } 
	return FALSE; 
} 

// CB for the set dialog
BOOL CALLBACK SetDialogProc(HWND hwndDlg, UINT message, WPARAM wParam, LPARAM lParam)
{
    switch(message)
    {
        case WM_INITDIALOG:
		{
			for (int i = 1; i < NUM_MODULES; i++)
			{
				if (SelectedTab == T_INSTRUMENT && i == M_ACC)
					SendDlgItemMessage(hwndDlg, IDC_SET_UNIT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"");
				else
					SendDlgItemMessage(hwndDlg, IDC_SET_UNIT, CB_ADDSTRING, (WPARAM)0, (LPARAM)(UnitName[i]));
			}
			SendDlgItemMessage(hwndDlg, IDC_SET_UNIT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);
			SetUnit = 1;
			return TRUE;
		}
        case WM_COMMAND:
		{
			if (((HWND)lParam) == GetDlgItem(hwndDlg, IDC_SET_UNIT))
			{
				if (HIWORD(wParam) == CBN_SELCHANGE)
				{
					SetUnit = 1+SendDlgItemMessage(hwndDlg, IDC_SET_UNIT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
				}
				return TRUE;
			}
            switch(LOWORD(wParam))
            {
                case IDOK:
					if (SelectedTab == T_INSTRUMENT && SetUnit == M_ACC)
						EndDialog(hwndDlg, IDCANCEL);
					else
						EndDialog(hwndDlg, IDOK);
                break;
                case IDCANCEL:
                    EndDialog(hwndDlg, IDCANCEL);
                break;
            }
			break;
		}
        default:
            return FALSE;
    }
    return TRUE;
}

VOID CALLBACK TimerProc(HWND hwnd, UINT uMsg, UINT_PTR idEvent, DWORD dwTime)
{
	for (int i = 0; i < 16; i++)
		UpdateVoiceDisplay(i);
}

VOID CALLBACK BackupTimerProc(HWND hwnd, UINT uMsg, UINT_PTR idEvent, DWORD dwTime)
{
	Go4kVSTi_SavePatch("C:\\4klang.4kp");
}

void Go4kVSTiGUI_Create(HINSTANCE hInst)
{
	timer = 1;
	backupTimer = 2;
	timerID = 0;
	backupTimerID = 0;
	DialogWnd = NULL;
	SelectedIUnit = 0;
	SelectedGUnit = 0;
	SelectedTab = T_INSTRUMENT;
	SelectedInstrument = 0;
	SetUnit = -1;
	hInstance = hInst;
	InitCommonControls();
	DialogWnd = CreateDialog(hInst,MAKEINTRESOURCE(IDD_GO4KVSTIDIALOG),NULL,(DLGPROC)MainDialogProc);

	// tab dialogs
	HWND tmp = GetDlgItem(DialogWnd, IDC_MODULE_SETTINGS);
	for (int i = 0; i < NUM_MODULES; i++)
	{	
		ModuleWnd[M_NONE+i] = CreateDialog(hInst,MAKEINTRESOURCE(IDD_NONE+i),tmp,(DLGPROC)ModuleSettingsDialogProc);
		MoveWindow(ModuleWnd[M_NONE+i], 7, 43, 450, 280, true);
		ShowWindow(ModuleWnd[M_NONE+i], SW_HIDE);
	}

	tmp = GetDlgItem(DialogWnd, IDC_MAINTAB);
	ScrollWnd = CreateDialog(hInst,MAKEINTRESOURCE(IDD_SCROLLWINDOW),tmp,(DLGPROC)ScrollDialogProc);
	MoveWindow(ScrollWnd, 3, 23, 392, 410, true);
	ShowWindow(ScrollWnd, SW_SHOW);

	tmp = ScrollWnd;
	RECT rect;
	// instrument stack dialog
	TabWnd[T_INSTRUMENT] = CreateDialogA(hInst,MAKEINTRESOURCE(IDD_INSTRUMENT_STACK),tmp,(DLGPROC)StackDialogProc);
	GetWindowRect(TabWnd[T_INSTRUMENT], &rect);
	int virtualsizey = rect.bottom - rect.top;
	MoveWindow(TabWnd[T_INSTRUMENT], 5, 0, 365, virtualsizey, true);
	ShowWindow(TabWnd[T_INSTRUMENT], SW_SHOW);
	EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_UNIT1), false);
	ShowWindow(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), true);
	
	// global stack dialog
	TabWnd[T_GLOBAL] = CreateDialog(hInst,MAKEINTRESOURCE(IDD_GLOBAL_STACK),tmp,(DLGPROC)StackDialogProc);
	MoveWindow(TabWnd[T_GLOBAL], 5, 0, 365, virtualsizey, true);
	ShowWindow(TabWnd[T_GLOBAL], SW_HIDE);
	EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_UNIT1), false);
	ShowWindow(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), false);

	// setup scrollwindow
	GetWindowRect(ScrollWnd, &rect);
	int sizey = rect.bottom - rect.top;
	SCROLLINFO sinfo;
	sinfo.cbSize = sizeof(SCROLLINFO);
	sinfo.fMask = SIF_PAGE | SIF_RANGE | SIF_POS;
	sinfo.nPos = 0;
	sinfo.nPage = sizey;
	sinfo.nMin = 0;
	sinfo.nMax = virtualsizey+2;
	SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true); 

	for (int i=0; i<MAX_INSTRUMENTS; i++)
		InstrumentScrollPos[i] = 0;
	GlobalScrollPos = 0;

	Go4kVSTi_Init();
	SynthObjP = Go4kVSTi_GetSynthObject();
	UpdateControls(SelectedInstrument);

	timerID = SetTimer(DialogWnd, timer, 200, TimerProc);
	backupTimerID = SetTimer(DialogWnd, backupTimer, 60*1000, BackupTimerProc);

	memset(UnitCopyBuffer, 0, MAX_UNIT_SLOTS);

	// set fst instrument elements
	const char* idest[] = {"Local", "Instrument  1", "Instrument  2", "Instrument  3", "Instrument  4", "Instrument  5", "Instrument  6", "Instrument  7", "Instrument  8",
				"Instrument  9", "Instrument 10", "Instrument 11", "Instrument 12", "Instrument 13", "Instrument 14", "Instrument 15", "Instrument 16", "Global" };
	for (int i = 0; i < 18; i++)
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)(idest[i]));

	InitDone = true;
}

void Go4kVSTiGUI_Show(int showCommand)
{
	ShowWindow(DialogWnd, showCommand);
}

void Go4kVSTiGUI_Destroy()
{
	Go4kVSTi_SavePatch("C:\\4klang.4kp");
/*
	if (MessageBox(NULL,"If you didnt save your 4klang patch yet, this information will be lost.\n"
					"Do you want to save the patch now?",
					"Reminder",
					MB_YESNO) == IDYES)
	{
		char lpstrFilter[256] =
		{'4','k','l','a','n','g',' ','P','a','t','c','h', 0 ,
		'*','.','4','k','p', 0, 
		0};
		char lpstrFile[4096] = "*.4kp";
		char lpstrDirectory[4096];
		GetCurrentDirectory(4096, lpstrDirectory);

		OPENFILENAME ofn =
		{
			sizeof(OPENFILENAME),
			DialogWnd,
			GetModuleHandle(NULL),
			lpstrFilter,
			NULL,
			0,
			1,
			lpstrFile,
			256,
			NULL,
			0,
			lpstrDirectory,
			NULL,
			OFN_OVERWRITEPROMPT,
			0,
			0,
			"4kp",
			0,
			NULL,
			NULL		
		};

		if (GetSaveFileName(&ofn))
		{
			Go4kVSTi_SavePatch(ofn.lpstrFile);
		}
	}	
*/
	KillTimer(DialogWnd, timerID);
	KillTimer(DialogWnd, backupTimerID);
	DestroyWindow(DialogWnd);
}

void LinkInstrumentUnit(int selectedInstrument, int selectedIUnit)
{
	for (int i = 0; i < 16; i++)
	{	
		if (LinkToInstrument[i] == selectedInstrument)
		{
			if (SynthObjP->InstrumentValues[selectedInstrument][selectedIUnit][0] == M_DLL)
			{
				DLL_valP ds = (DLL_valP)SynthObjP->InstrumentValues[selectedInstrument][selectedIUnit];
				DLL_valP dl = (DLL_valP)SynthObjP->InstrumentValues[i][selectedIUnit];
				dl->id			= ds->id;
				dl->pregain		= ds->pregain;
				dl->dry			= ds->dry;
				dl->feedback	= ds->feedback;
				dl->damp		= ds->damp;
				dl->freq		= ds->freq;
				dl->depth		= ds->depth;
				dl->guidelay	= ds->guidelay;
				dl->synctype	= ds->synctype;
				dl->leftreverb	= ds->leftreverb;
				dl->reverb		= ds->reverb;
			}
			else if (SynthObjP->InstrumentValues[selectedInstrument][selectedIUnit][0] == M_GLITCH)
			{
				GLITCH_valP ds = (GLITCH_valP)SynthObjP->InstrumentValues[selectedInstrument][selectedIUnit];
				GLITCH_valP dl = (GLITCH_valP)SynthObjP->InstrumentValues[i][selectedIUnit];
				dl->id			= ds->id;
				dl->active		= ds->active;
				dl->dry			= ds->dry;
				dl->dsize		= ds->dsize;
				dl->dpitch		= ds->dpitch;
				dl->guidelay	= ds->guidelay;
			}
			else
			{
				memcpy(SynthObjP->InstrumentValues[i][selectedIUnit], 
						SynthObjP->InstrumentValues[selectedInstrument][selectedIUnit], 
						MAX_UNIT_SLOTS);
			}
		}
	}
	Go4kVSTi_UpdateDelayTimes();
}

//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////
//																		//
//                          Button Events								//
//																		//
//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////

void SetButtonParams(int uid, BYTE* val, WPARAM id, LPARAM lParam)
{
	DWORD res;
	if (uid == M_VCO)
	{
		VCO_valP v = (VCO_valP)val;
		ButtonGroupChanged(IDC_VCO_SINE, IDC_VCO_GATE, LOWORD(id), M_VCO, res);
		if (res)
		{
			v->flags = 0;
			if (SendDlgItemMessage(ModuleWnd[M_VCO], IDC_VCO_LFO, BM_GETCHECK, 0, 0)==BST_CHECKED)
				v->flags |= VCO_LFO;
			if (SendDlgItemMessage(ModuleWnd[M_VCO], IDC_VCO_STEREO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->flags |= VCO_STEREO;

			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE_VAL), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE_VAL), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE_VAL), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR_VAL), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE_VAL), true);
			EnableButtonGroup(IDC_VCO_GATE1, IDC_VCO_GATE16, M_VCO, false);
			if (IDC_VCO_GATE == res)
			{
				v->flags |= VCO_GATE;
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR_VAL), false);
				EnableButtonGroup(IDC_VCO_GATE1, IDC_VCO_GATE16, M_VCO, true);
				// adjust gate bits stored in color and shape with current selection of checkboxes
				WORD gatebits;
				GetCheckboxGroupBitmask(IDC_VCO_GATE1, IDC_VCO_GATE16, IDC_VCO_GATE1, M_VCO, gatebits, res);
				v->gate = gatebits & 0xff;
				v->color = (gatebits >> 8) & 0xff;
				// adjust color and shape sliders
				InitSliderCenter(ModuleWnd[M_VCO], IDC_VCO_COLOR, 0, 128, v->color);
			}
			else
			{
				// set color at least to valid range
				if (v->color > 128) v->color = 128;
				InitSliderCenter(ModuleWnd[M_VCO], IDC_VCO_COLOR, 0, 128, v->color);
			}
			if (IDC_VCO_SINE == res)
			{
				v->flags |= VCO_SINE;
			}
			if (IDC_VCO_TRISAW == res)
			{
				v->flags |= VCO_TRISAW;
			}
			if (IDC_VCO_PULSE == res)
			{
				v->flags |= VCO_PULSE;
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE_VAL), false);
			}
			if (IDC_VCO_NOISE == res)
			{
				v->flags |= VCO_NOISE;
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE_VAL), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE_VAL), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE_VAL), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR_VAL), false);
			}			
		}
		if (LOWORD(id) == IDC_VCO_LFO)
		{
			if (SendDlgItemMessage(ModuleWnd[M_VCO], IDC_VCO_LFO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->flags |= VCO_LFO;
			else
				v->flags &= ~VCO_LFO;
		}
		if (LOWORD(id) == IDC_VCO_STEREO)
		{			
			if (SendDlgItemMessage(ModuleWnd[M_VCO], IDC_VCO_STEREO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->flags |= VCO_STEREO;
			else
				v->flags &= ~VCO_STEREO;
		}
		// gate bits changed?
		WORD gatebits;
		GetCheckboxGroupBitmask(IDC_VCO_GATE1, IDC_VCO_GATE16, LOWORD(id), M_VCO, gatebits, res);
		if (res)
		{
			v->gate = gatebits & 0xff;
			v->color = (gatebits >> 8) & 0xff;
			// adjust color slider
			InitSliderCenter(ModuleWnd[M_VCO], IDC_VCO_COLOR, 0, 128, v->color);
		}
		// update signalcount
		UpdateSignalCount(SelectedInstrument);
	}
	else if (uid == M_VCF)
	{
		VCF_valP v = (VCF_valP)val;
		ButtonGroupChanged(IDC_VCF_LOW, IDC_VCF_ALL, LOWORD(id), M_VCF, res);
		if (res)
		{
			int stereo = v->type & VCF_STEREO;
			if (res == IDC_VCF_LOW)
				v->type = stereo | VCF_LOWPASS;
			else if (res == IDC_VCF_HIGH)
				v->type = stereo | VCF_HIGHPASS;
			else if (res == IDC_VCF_BAND)
				v->type = stereo | VCF_BANDPASS;
			else if (res == IDC_VCF_NOTCH)
				v->type = stereo | VCF_BANDSTOP;
			else if (res == IDC_VCF_PEAK)
				v->type = stereo | VCF_PEAK;
			else if (res == IDC_VCF_ALL)
				v->type = stereo | VCF_ALLPASS;
		}
		if (LOWORD(id) == IDC_VCF_STEREO)
		{			
			if (SendDlgItemMessage(ModuleWnd[M_VCF], IDC_VCF_STEREO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->type |= VCF_STEREO;
			else
				v->type &= ~VCF_STEREO;
		}
		// update signalcount
		UpdateSignalCount(SelectedInstrument);
	}
	else if (uid == M_DST)
	{
		DST_valP v = (DST_valP)val;
		if (LOWORD(id) == IDC_DST_STEREO)
		{			
			if (SendDlgItemMessage(ModuleWnd[M_DST], IDC_DST_STEREO, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->stereo = VCF_STEREO;
			else
				v->stereo = 0;
		}
		// update signalcount
		UpdateSignalCount(SelectedInstrument);
	}
	else if (uid == M_DLL)
	{
		DLL_valP v = (DLL_valP)val;
		ButtonGroupChanged(IDC_DLL_DELAY, IDC_DLL_REVERB, LOWORD(id), M_DLL, res);
		if (res)
		{
			if (res == IDC_DLL_DELAY)
			{
				v->reverb = 0;
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_LEFTREVERB), SW_HIDE);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), SW_SHOW);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), SW_SHOW);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOSYNC), SW_SHOW);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_BPMSYNC), SW_SHOW);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOTESYNC), SW_SHOW);
			}
			else if (res == IDC_DLL_REVERB)
			{
				v->reverb = 1;
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_LEFTREVERB), SW_SHOW);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), SW_HIDE);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), SW_HIDE);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOSYNC), SW_HIDE);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_BPMSYNC), SW_HIDE);
				ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOTESYNC), SW_HIDE);
			}
		}
		ButtonGroupChanged(IDC_DLL_NOSYNC, IDC_DLL_NOTESYNC, LOWORD(id), M_DLL, res);
		if (res)
		{
			if (res == IDC_DLL_NOSYNC)
			{
				v->synctype = 0;
				EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), true);
				EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), true);
			}
			else if (res == IDC_DLL_BPMSYNC)
			{
				v->synctype = 1;
				EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), true);
				EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), true);
			}
			else if (res == IDC_DLL_NOTESYNC)
			{
				v->synctype = 2;
				EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), false);
				EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), false);
			}
		}
		if (LOWORD(id) == IDC_DLL_LEFTREVERB)
		{			
			if (SendDlgItemMessage(ModuleWnd[M_DLL], IDC_DLL_LEFTREVERB, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->leftreverb = 1;
			else
				v->leftreverb = 0;
		}
		UpdateDelayTimes(v);
	}
	else if (uid == M_FOP)
	{
		// invalidate stack first
		if (SelectedTab == T_INSTRUMENT)
		{
			SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
		}
		else
		{
			SynthObjP->GlobalSignalValid = 0;
		}
		FOP_valP v = (FOP_valP)val;
		ButtonGroupChanged(IDC_FOP_POP, IDC_FOP_MULP2, LOWORD(id), M_FOP, res);
		if (res)
		{
			v->flags = 1 + res - IDC_FOP_POP;
		}		
		// update signalcount
		UpdateSignalCount(SelectedInstrument);
	}
	else if (uid == M_FST)
	{
		FST_valP v = (FST_valP)val;
		// changed instrument
		if (LOWORD(id) == IDC_FST_DESTINATION_INSTRUMENT)
		{
			int inst = SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0) - 1;
			if (v->dest_stack != inst)
			{
				v->dest_stack = inst;

				// fill unit combobox with current units and set to nothing
				BYTE *units;
				if (SelectedTab == T_INSTRUMENT)
				{
					if (v->dest_stack == -1)
						units = SynthObjP->InstrumentValues[SelectedInstrument][0];
					else if (v->dest_stack == MAX_INSTRUMENTS)
						units = SynthObjP->GlobalValues[0];
					else
						units = SynthObjP->InstrumentValues[v->dest_stack][0];
				}
				else
				{
					if (v->dest_stack == -1)
						units = SynthObjP->GlobalValues[0];
					else if (v->dest_stack == MAX_INSTRUMENTS)
						units = SynthObjP->GlobalValues[0];
					else
						units = SynthObjP->InstrumentValues[v->dest_stack][0];
				}

				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"- Nothing Selected -");
				for (int i = 0; i < MAX_UNITS; i++)
				{
					char unitname[128], unitname2[128];
					sprintf(unitname, "%d: %s", i+1, GetUnitString(&units[i*MAX_UNIT_SLOTS], unitname2));
					SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_ADDSTRING, (WPARAM)0, (LPARAM)unitname);
				}
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);
				v->dest_unit = -1;

				// set slot combobox to nothing and disable
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"- Nothing Selected -");
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);
				EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT), false);
				v->dest_slot = -1;
				v->dest_id = -1;
			}
		}
		else if (LOWORD(id) == IDC_FST_DESTINATION_UNIT)
		{
			int unit = SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0) - 1;
			if (v->dest_unit != unit)
			{
				v->dest_unit = unit;

				// fill slot combobox with current targets and set to nothing
				BYTE *slots;
				if (SelectedTab == T_INSTRUMENT)
				{
					if (v->dest_stack == -1)
						slots = SynthObjP->InstrumentValues[SelectedInstrument][unit];
					else if (v->dest_stack == MAX_INSTRUMENTS)
						slots = SynthObjP->GlobalValues[unit];
					else
						slots = SynthObjP->InstrumentValues[v->dest_stack][unit];
				}
				else
				{
					if (v->dest_stack == -1)
						slots = SynthObjP->GlobalValues[unit];
					else if (v->dest_stack == MAX_INSTRUMENTS)
						slots = SynthObjP->GlobalValues[unit];
					else
						slots = SynthObjP->InstrumentValues[v->dest_stack][unit];
				}

				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"- Nothing Selected -");
				for (int i = 0; i < 8; i++)
				{
					SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_ADDSTRING, (WPARAM)0, (LPARAM)UnitModulationTargetNames[slots[0]][i]);
				}
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);
				v->dest_slot = -1;
				v->dest_id = -1;

				if (v->dest_unit == -1)
					EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT), false);
				else
					EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT), true);
			}
		}
		else if (LOWORD(id) == IDC_FST_DESTINATION_SLOT)
		{
			int slot = SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0) - 1;
			v->dest_slot = slot;

			// fill slot combobox with current targets and set to nothing
			BYTE *slots;
			if (SelectedTab == T_INSTRUMENT)
			{
				if (v->dest_stack == -1)
					slots = SynthObjP->InstrumentValues[SelectedInstrument][v->dest_unit];
				else if (v->dest_stack == MAX_INSTRUMENTS)
					slots = SynthObjP->GlobalValues[v->dest_unit];
				else
					slots = SynthObjP->InstrumentValues[v->dest_stack][v->dest_unit];
			}
			else
			{
				if (v->dest_stack == -1)
					slots = SynthObjP->GlobalValues[v->dest_unit];
				else if (v->dest_stack == MAX_INSTRUMENTS)
					slots = SynthObjP->GlobalValues[v->dest_unit];
				else
					slots = SynthObjP->InstrumentValues[v->dest_stack][v->dest_unit];
			}

			v->dest_id = slots[0];
		}
		ButtonGroupChanged(IDC_FST_SET, IDC_FST_MUL, LOWORD(id), M_FST, res);
		if (res)
		{
			int pop = v->type & FST_POP;
			if (res == IDC_FST_SET)
				v->type = FST_SET;
			else if (res == IDC_FST_ADD)
				v->type = FST_ADD;
			else if (res == IDC_FST_MUL)
				v->type = FST_MUL;
			if (pop)
				v->type |= FST_POP;
		}
		if (LOWORD(id) == IDC_FST_POP)
		{			
			if (SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_POP, BM_GETCHECK, 0, 0)==BST_CHECKED) 
				v->type |= FST_POP;
			else
				v->type &= ~FST_POP;
		}
		// update signalcount
		UpdateSignalCount(SelectedInstrument);
	}
	else if (uid == M_ACC)
	{
		ACC_valP v = (ACC_valP)val;
		ButtonGroupChanged(IDC_ACC_OUT, IDC_ACC_AUX, LOWORD(id), M_ACC, res);
		if (res)
		{
			if (res == IDC_ACC_OUT)
				v->flags = ACC_OUT;
			else
				v->flags = ACC_AUX;
		}
	}
}

bool ButtonPressed(WPARAM id, LPARAM lParam)
{
	// instrument tab is active
	if (SelectedTab == T_INSTRUMENT)
	{
		UnitID uid = (UnitID)(SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit][0]);
		SetButtonParams(uid, SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit], id, lParam);
		char unitname[128];
		GetUnitString(SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit], unitname);
		SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_UNIT1+SelectedIUnit*6), unitname);
		LinkInstrumentUnit(SelectedInstrument, SelectedIUnit);
	}
	else if (SelectedTab == T_GLOBAL)
	{		
		UnitID uid = (UnitID)(SynthObjP->GlobalValues[SelectedGUnit][0]);
		SetButtonParams(uid, SynthObjP->GlobalValues[SelectedGUnit], id, lParam);
		char unitname[128];
		GetUnitString(SynthObjP->GlobalValues[SelectedGUnit], unitname);
		SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_UNIT1+SelectedGUnit*6), unitname);
	}
	return TRUE;
}

//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////
//																		//
//                        Scrollbar Events								//
//																		//
//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////

void SetSliderParams(int uid, BYTE* val, LPARAM lParam)
{
	if (uid == M_ENV)
	{
		ENV_valP v = (ENV_valP)val;
		// attack
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_ENV_ATTACK))
		{
			UpdateSliderValue(IDC_ENV_ATTACK, v->attac);
		}
		//decay
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_ENV_DECAY))
		{
			UpdateSliderValue(IDC_ENV_DECAY, v->decay);
		}
		// sustain
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_ENV_SUSTAIN))
		{
			UpdateSliderValue(IDC_ENV_SUSTAIN, v->sustain);
		}
		// release
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_ENV_RELEASE))
		{
			UpdateSliderValue(IDC_ENV_RELEASE, v->release);
		}
		// gain
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_ENV_GAIN))
		{
			UpdateSliderValue(IDC_ENV_GAIN, v->gain);
		}
	}
	else if (uid == M_VCO)
	{
		VCO_valP v = (VCO_valP)val;
		// transpose
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCO_TRANSPOSE))
		{
			UpdateSliderValueCenter(IDC_VCO_TRANSPOSE, v->transpose);
		}
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCO_DETUNE))
		{
			UpdateSliderValueCenter(IDC_VCO_DETUNE, v->detune);
		}
		// phaseofs
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCO_PHASE))
		{
			UpdateSliderValue(IDC_VCO_PHASE, v->phaseofs);
		}
		// color
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCO_COLOR))
		{
			UpdateSliderValueCenter(IDC_VCO_COLOR, v->color);
			// gate bits
			WORD gatebits = ((WORD)v->color << 8) | (WORD)v->gate ;
			SetCheckboxGroupBitmask(IDC_VCO_GATE1, IDC_VCO_GATE16, M_VCO, gatebits);
		}
		// shape
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCO_SHAPE))
		{
			v->shape = SendMessage(GetDlgItem(ModuleWnd[uid], IDC_VCO_SHAPE), TBM_GETPOS, 0, 0);
			if (v->shape == 0)
				v->shape = 1;
			if (v->shape == 128)
				v->shape = 127;
			sprintf(SliderValTxt, "%d", v->shape-64);
			SetWindowText(GetDlgItem(ModuleWnd[uid], IDC_VCO_SHAPE_VAL), SliderValTxt);
		}
		// gain
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCO_GAIN))
		{
			UpdateSliderValue(IDC_VCO_GAIN, v->gain);
		}
	}
	else if (uid == M_VCF)
	{
		VCF_valP v = (VCF_valP)val;
		// frequency
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCF_FREQUENCY))
		{
			UpdateSliderValue(IDC_VCF_FREQUENCY, v->freq);
		}
		// resonance
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_VCF_RESONANCE))
		{
			UpdateSliderValue(IDC_VCF_RESONANCE, v->res);
		}
	}
	else if (uid == M_DST)
	{
		DST_valP v = (DST_valP)val;
		// drive
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DST_DRIVE))
		{
			v->drive = SendMessage(GetDlgItem(ModuleWnd[uid], IDC_DST_DRIVE), TBM_GETPOS, 0, 0);
			if (v->drive == 0)
				v->drive = 1;
			if (v->drive == 128)
				v->drive = 127;
			sprintf(SliderValTxt, "%d", v->drive-64);
			SetWindowText(GetDlgItem(ModuleWnd[uid], IDC_DST_DRIVE_VAL), SliderValTxt);
		}		
		// snhfreq
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DST_SNH))
		{
			UpdateSliderValue(IDC_DST_SNH, v->snhfreq);
		}	
	}
	else if (uid == M_DLL)
	{
		DLL_valP v = (DLL_valP)val;
		// pregain
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_PREGAIN))
		{
			UpdateSliderValue(IDC_DLL_PREGAIN, v->pregain);
		}
		// dry
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_DRY))
		{
			UpdateSliderValue(IDC_DLL_DRY, v->dry);
		}		
		// feedback
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_FEEDBACK))
		{
			UpdateSliderValue(IDC_DLL_FEEDBACK, v->feedback);
		}
		// frequency
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_FREQUENCY))
		{
			UpdateSliderValue(IDC_DLL_FREQUENCY, v->freq);
		}
		// depth
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_DEPTH))
		{
			UpdateSliderValue(IDC_DLL_DEPTH, v->depth);
		}
		// damp
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_DAMP))
		{
			UpdateSliderValue(IDC_DLL_DAMP, v->damp);
		}
		// delay
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_DLL_DTIME))
		{
			v->guidelay = SendMessage(GetDlgItem(ModuleWnd[uid], IDC_DLL_DTIME), TBM_GETPOS, 0, 0);
			UpdateDelayTimes(v);
		}
	}
	else if (uid == M_FST)
	{
		FST_valP v = (FST_valP)val;
		// gain
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_FST_AMOUNT))
		{
			UpdateSliderValueCenter2(IDC_FST_AMOUNT, v->amount);
		}
	}
	else if (uid == M_PAN)
	{
		PAN_valP v = (PAN_valP)val;
		// panning
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_PAN_PANNING))
		{
			UpdateSliderValueCenter(IDC_PAN_PANNING, v->panning);
		}
	}
	else if (uid == M_OUT)
	{
		OUT_valP v = (OUT_valP)val;
		// gain
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_OUT_GAIN))
		{
			UpdateSliderValue(IDC_OUT_GAIN, v->gain);
		}		
		// auxsend
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_OUT_AUXSEND))
		{
			UpdateSliderValue(IDC_OUT_AUXSEND, v->auxsend);
		}
	}
	else if (uid == M_FLD)
	{
		FLD_valP v = (FLD_valP)val;
		// panning
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_FLD_VALUE))
		{
			UpdateSliderValueCenter(IDC_FLD_VALUE, v->value);
		}
	}
	else if (uid == M_GLITCH)
	{
		GLITCH_valP v = (GLITCH_valP)val;
		// active
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_GLITCH_ACTIVE))
		{
			UpdateSliderValue(IDC_GLITCH_ACTIVE, v->active);
		}
		// dry
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_GLITCH_DRY))
		{
			UpdateSliderValue(IDC_GLITCH_DRY, v->dry);
		}
		// delta size
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_GLITCH_DSIZE))
		{
			UpdateSliderValue(IDC_GLITCH_DSIZE, v->dsize);
		}
		// delta pitch
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_GLITCH_DPITCH))
		{
			UpdateSliderValue(IDC_GLITCH_DPITCH, v->dpitch);
		}
		// delay
		if ((HWND)lParam == GetDlgItem(ModuleWnd[uid], IDC_GLITCH_DTIME))
		{
			v->guidelay = SendMessage(GetDlgItem(ModuleWnd[uid], IDC_GLITCH_DTIME), TBM_GETPOS, 0, 0);
			UpdateDelayTimes(v);
		}
	}
}

bool ScrollbarChanged(HWND hwndDlg, WPARAM wParam, LPARAM lParam)
{
	// instrument tab is active
	if (SelectedTab == T_INSTRUMENT)
	{
		UnitID uid = (UnitID)(SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit][0]);
		SetSliderParams(uid, SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit], lParam);
		char unitname[128];
		GetUnitString(SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit], unitname);
		SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_UNIT1+SelectedIUnit*6), unitname);
		LinkInstrumentUnit(SelectedInstrument, SelectedIUnit);
	}
	else if (SelectedTab == T_GLOBAL)
	{		
		UnitID uid = (UnitID)(SynthObjP->GlobalValues[SelectedGUnit][0]);
		SetSliderParams(uid, SynthObjP->GlobalValues[SelectedGUnit], lParam);
		char unitname[128];
		GetUnitString(SynthObjP->GlobalValues[SelectedGUnit], unitname);
		SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_UNIT1+SelectedGUnit*6), unitname);
	}
	return TRUE;
}

//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////
//																		//
//                       Stack Button Events							//
//																		//
//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////

void CheckModulations(int unit1, int unit2 = -1)
{
	// a instrument stack changed
	if (SelectedTab == T_INSTRUMENT)
	{
		// look in all instruments if a store unit had its target on one of the changed units
		for (int i = 0; i < MAX_INSTRUMENTS; i++)
		{
			for (int u = 0; u < MAX_UNITS; u++)
			{
				if (SynthObjP->InstrumentValues[i][u][0] == M_FST)
				{
					FST_valP v = (FST_valP)(SynthObjP->InstrumentValues[i][u]);
					// the store points to global, so skip
					if (v->dest_stack == MAX_INSTRUMENTS)
						continue;
					
					int target_inst;
					if (v->dest_stack == -1) 
						target_inst = i;
					else
						target_inst = v->dest_stack;

					// if the store points to the current instrument
					if (target_inst == SelectedInstrument)
					{
						// a up/down process 
						if (unit2 != -1)
						{
							// if the target unit was unit1 or unit2 realign target unit
							if (v->dest_unit == unit1)
							{
								v->dest_unit = unit2;
							}
							else if (v->dest_unit == unit2)
							{
								v->dest_unit = unit1;
							}
						}
						// a set/reset process
						else
						{
							// if the target unit was unit1 reset store
							if (v->dest_unit == unit1)
							{
								char text[128];
								sprintf(text, "Instrument %d had a modulation target in the changed unit\nIt's no longer valid and will be reset", i);
								MessageBox(DialogWnd, text, "Info", MB_OK);
								v->dest_unit = -1;
								v->dest_slot = -1;
								v->dest_id = -1;
							}
						}
					}
				}
			}
		}
		// now check the global slots
		for (int u = 0; u < MAX_UNITS; u++)
		{
			if (SynthObjP->GlobalValues[u][0] == M_FST)
			{
				FST_valP v = (FST_valP)(SynthObjP->GlobalValues[u]);
				// the store points to global, so skip
				if (v->dest_stack == -1 || v->dest_stack == MAX_INSTRUMENTS)
					continue;

				// if the store points to the current instrument
				if (v->dest_stack == SelectedInstrument)
				{
					// a up/down process 
					if (unit2 != -1)
					{
						// if the target unit was unit1 or unit2 realign target unit
						if (v->dest_unit == unit1)
						{
							v->dest_unit = unit2;
						}
						else if (v->dest_unit == unit2)
						{
							v->dest_unit = unit1;
						}
					}
					// a set/reset process
					else
					{
						// if the target unit was unit1 reset store
						if (v->dest_unit == unit1)
						{
							char text[128];
							sprintf(text, "Global had a modulation target in the changed unit\nIt's no longer valid and will be reset");
							MessageBox(DialogWnd, text, "Info", MB_OK);
							v->dest_unit = -1;
							v->dest_slot = -1;
							v->dest_id = -1;
						}
					}
				}
			}
		}
	}
	// the global stack changed
	else
	{
		// look in all instruments if a store unit had its target on a global unit
		for (int i = 0; i < MAX_INSTRUMENTS; i++)
		{
			for (int u = 0; u < MAX_UNITS; u++)
			{
				if (SynthObjP->InstrumentValues[i][u][0] == M_FST)
				{
					FST_valP v = (FST_valP)(SynthObjP->InstrumentValues[i][u]);
					// the store doesnt point to global, so skip
					if (v->dest_stack != MAX_INSTRUMENTS)
						continue;

					// a up/down process 
					if (unit2 != -1)
					{
						// if the target unit was unit1 or unit2 realign target unit
						if (v->dest_unit == unit1)
						{
							v->dest_unit = unit2;
						}
						else if (v->dest_unit == unit2)
						{
							v->dest_unit = unit1;
						}
					}
					// a set/reset process
					else
					{
						// if the target unit was unit1 reset store
						if (v->dest_unit == unit1)
						{
							char text[128];
							sprintf(text, "Instrument %d had a modulation target in the changed unit\nIt's no longer valid and will be reset", i);
							MessageBox(DialogWnd, text, "Info", MB_OK);
							v->dest_unit = -1;
							v->dest_slot = -1;
							v->dest_id = -1;
						}
					}
				}
			}
		}
		// now check the global slots
		for (int u = 0; u < MAX_UNITS; u++)
		{
			if (SynthObjP->GlobalValues[u][0] == M_FST)
			{
				FST_valP v = (FST_valP)(SynthObjP->GlobalValues[u]);
				// the store points to global, so check
				if (v->dest_stack == -1 || v->dest_stack == MAX_INSTRUMENTS)
				{
					// a up/down process 
					if (unit2 != -1)
					{
						// if the target unit was unit1 or unit2 realign target unit
						if (v->dest_unit == unit1)
						{
							v->dest_unit = unit2;
						}
						else if (v->dest_unit == unit2)
						{
							v->dest_unit = unit1;
						}
					}
					// a set/reset process
					else
					{
						// if the target unit was unit1 reset store
						if (v->dest_unit == unit1)
						{
							char text[128];
							sprintf(text, "Global had a modulation target in the changed unit\nIt's no longer valid and will be reset");
							MessageBox(DialogWnd, text, "Info", MB_OK);
							v->dest_unit = -1;
							v->dest_slot = -1;
							v->dest_id = -1;
						}
					}
				}
			}
		}
	}
}

bool StackButtonPressed(WPARAM id)
{
	int res;

	// Instrument Stack Buttons
	if (SelectedTab == T_INSTRUMENT)
	{
		// check unit click
		InterleavedButtonGroupChanged(IDC_ISTACK_UNIT1, MAX_UNITS, LOWORD(id), res);
		if (res >= 0)
		{
			SelectedIUnit = res;
			UpdateModuleParamWindow(T_INSTRUMENT, res);
		}
		
		// check down click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_ISTACK_DOWN1+i*6;
			if (LOWORD(id) == tid)
			{
				SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), false);
				Go4kVSTi_FlipInstrumentSlots(SelectedInstrument, i, i+1);
				CheckModulations(i, i+1);
				LinkInstrumentUnit(SelectedInstrument, i);
				LinkInstrumentUnit(SelectedInstrument, i+1);
				UpdateControls(SelectedInstrument);
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), true);
			}			
		}

		// check up click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_ISTACK_UP1+i*6;
			if (LOWORD(id) == tid)
			{
				SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), false);
				Go4kVSTi_FlipInstrumentSlots(SelectedInstrument, i, i-1);
				CheckModulations(i, i-1);
				LinkInstrumentUnit(SelectedInstrument, i);
				LinkInstrumentUnit(SelectedInstrument, i-1);
				UpdateControls(SelectedInstrument);
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), true);
			}
		}

		// check set click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_ISTACK_SET1+i*6;
			if (LOWORD(id) == tid)
			{
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), false);
				if (IDOK == DialogBox(hInstance, MAKEINTRESOURCE(IDD_SET_DIALOG), DialogWnd, (DLGPROC)SetDialogProc))
				{
					if (SetUnit != SynthObjP->InstrumentValues[SelectedInstrument][i][0])
					{
						SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
						Go4kVSTi_InitInstrumentSlot(SelectedInstrument, i, SetUnit);
						CheckModulations(i);
						SelectedIUnit = i;
						LinkInstrumentUnit(SelectedInstrument, i);
						UpdateControls(SelectedInstrument);
					}
				}
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), true);
				return TRUE;
			}
		}

		// check reset click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_ISTACK_RESET1+i*6;
			if (LOWORD(id) == tid)
			{
				SynthObjP->InstrumentSignalValid[SelectedInstrument] = 0;
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), false);			
				Go4kVSTi_ClearInstrumentSlot(SelectedInstrument, i);
				LinkInstrumentUnit(SelectedInstrument, i);
				CheckModulations(i);

				UpdateControls(SelectedInstrument);
				EnableWindow(GetDlgItem(TabWnd[T_INSTRUMENT], tid), true);
			}			
		}
	}
	else if (SelectedTab == T_GLOBAL)
	{
		// check unit click
		InterleavedButtonGroupChanged(IDC_GSTACK_UNIT1, MAX_UNITS, LOWORD(id), res);
		if (res >= 0)
		{
			SelectedGUnit = res;
			UpdateModuleParamWindow(T_GLOBAL, res);
		}

		// check down click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_GSTACK_DOWN1+i*6;
			if (LOWORD(id) == tid)
			{
				SynthObjP->GlobalSignalValid = 0;
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), false);
				Go4kVSTi_FlipGlobalSlots(i, i+1);
				CheckModulations(i, i+1);
				UpdateControls(SelectedInstrument);
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), true);
			}
		}

		// check up click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_GSTACK_UP1+i*6;
			if (LOWORD(id) == tid)
			{
				SynthObjP->GlobalSignalValid = 0;
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), false);
				Go4kVSTi_FlipGlobalSlots(i, i-1);
				CheckModulations(i, i-1);
				UpdateControls(SelectedInstrument);
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), true);
			}
		}

		// check set click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_GSTACK_SET1+i*6;
			if (LOWORD(id) == tid)
			{
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), false);
				if (IDOK == DialogBox(hInstance, MAKEINTRESOURCE(IDD_SET_DIALOG), DialogWnd, (DLGPROC)SetDialogProc))
				{
					if (SetUnit != SynthObjP->GlobalValues[i][0])
					{
						SynthObjP->GlobalSignalValid = 0;
						Go4kVSTi_InitGlobalSlot(i, SetUnit);
						CheckModulations(i);
						SelectedGUnit = i;
						UpdateControls(SelectedInstrument);
					}
				}
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), true);
			}
		}

		// check reset click
		for (int i = 0; i < MAX_UNITS; i++)
		{
			WORD tid = IDC_GSTACK_RESET1+i*6;
			if (LOWORD(id) == tid)
			{
				SynthObjP->GlobalSignalValid = 0;
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), false);
				Go4kVSTi_ClearGlobalSlot(i);
				CheckModulations(i);
				UpdateControls(SelectedInstrument);
				EnableWindow(GetDlgItem(TabWnd[T_GLOBAL], tid), true);
			}
		}
	}
	return TRUE;
}

//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////
//																		//
//                        TabChange Event 								//
//																		//
//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////

void TabChanged(int index)
{
	for (int i = 0; i < NUM_TABS; i++)
	{
		if (i == index)
			ShowWindow(TabWnd[i], SW_SHOW);		
		else
			ShowWindow(TabWnd[i], SW_HIDE);			
	}
	if (index == T_INSTRUMENT)
	{
		ShowWindow(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), true);
		ShowWindow(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), false);
		// reset scroll pos
		int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
		SCROLLINFO sinfo;
		sinfo.cbSize = sizeof(SCROLLINFO);
		sinfo.fMask = SIF_POS;
		sinfo.nPos = InstrumentScrollPos[SelectedInstrument];
		SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);				
		ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);		
	}
	if (index == T_GLOBAL)
	{
		ShowWindow(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), false);
		ShowWindow(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), true);
		// reset scroll pos
		int oldpos = GetScrollPos(ScrollWnd, SB_VERT);
		SCROLLINFO sinfo;
		sinfo.cbSize = sizeof(SCROLLINFO);
		sinfo.fMask = SIF_POS;
		sinfo.nPos = GlobalScrollPos;
		SetScrollInfo(ScrollWnd, SB_VERT, &sinfo, true);
		ScrollWindow(ScrollWnd, 0, oldpos - sinfo.nPos, NULL, NULL);
	}

	SelectedTab = index;
	UpdateModuleParamWindow(SelectedTab, -1);
}

//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////
//																		//
//                        GUI Update Events								//
//																		//
//////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////

void UpdateSignalCount(int channel)
{
	//////////////////////////////////////////////////////////////
	//  instrument
	//////////////////////////////////////////////////////////////

	// disable for safety first
	SynthObjP->InstrumentSignalValid[channel] = 0;
	int valid = 1;
	int signalcount = 0;
	// check stack validity
	SynthObjP->HighestSlotIndex[channel] = 63;
	int highestSlotIndex = 0;	
	for (int i = 0; i < MAX_UNITS; i++)
	{
		if (SynthObjP->InstrumentValues[channel][i][0] != 0)
		{
			if (i >= highestSlotIndex)
				highestSlotIndex = i;
		}
		// check unit precondition
		int precond = UnitPreSignals[SynthObjP->InstrumentValues[channel][i][0]];
		// adjust for arithmetic unit (depending on mode)
		if (SynthObjP->InstrumentValues[channel][i][0] == M_FOP)
		{
			FOP_valP v = (FOP_valP)(SynthObjP->InstrumentValues[channel][i]);
			if (v->flags == FOP_LOADNOTE)
				precond = 0;
			if (v->flags == FOP_POP)
				precond = 1;
			if (v->flags == FOP_PUSH)
				precond = 1;
			if (v->flags == FOP_ADDP2)
				precond = 4;
			if (v->flags == FOP_MULP2)
				precond = 4;
		}
		// adjust for stereo in vcf (we need 2 signals for stereo)
		if (SynthObjP->InstrumentValues[channel][i][0] == M_VCF)
		{
			VCF_valP v = (VCF_valP)(SynthObjP->InstrumentValues[channel][i]);
			if (v->type & VCF_STEREO)
				precond += 1;
		}
		// adjust for stereo in dst (we need 2 signals for stereo)
		if (SynthObjP->InstrumentValues[channel][i][0] == M_DST)
		{
			DST_valP v = (DST_valP)(SynthObjP->InstrumentValues[channel][i]);
			if (v->stereo & VCF_STEREO)
				precond += 1;
		}
		if (signalcount < precond)
		{
			valid = 0;
			Go4kVSTi_ClearInstrumentWorkspace(channel);
			for (int j = i; j < MAX_UNITS; j++)
			{
				char txt[10];
				sprintf(txt, "%d", signalcount);
				SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_SIGNALCOUNT1+j*6), "inv");
			}
			char err[128];
			sprintf(err, "Signal INVALID! Unit %d needs >= %d signals", i+1, precond);
			SetWindowText(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), err);
			break;
		}
		// precondition ok
		else
		{
			signalcount += UnitPostSignals[SynthObjP->InstrumentValues[channel][i][0]];
			// adjust for arithmetic unit (depending on mode)
			if (SynthObjP->InstrumentValues[channel][i][0] == M_FOP)
			{
				FOP_valP v = (FOP_valP)(SynthObjP->InstrumentValues[channel][i]);
				if (v->flags == FOP_POP)
					signalcount--;
				if (v->flags == FOP_PUSH)
					signalcount++;
				if (v->flags == FOP_ADDP)
					signalcount--;
				if (v->flags == FOP_MULP)
					signalcount--;
				if (v->flags == FOP_ADDP2)
					signalcount-=2;
				if (v->flags == FOP_MULP2)
					signalcount-=2;
				if (v->flags == FOP_LOADNOTE)
					signalcount++;
			}
			// adjust for stereo in vco (2 signals for stereo)
			if (SynthObjP->InstrumentValues[channel][i][0] == M_VCO)
			{
				VCO_valP v = (VCO_valP)(SynthObjP->InstrumentValues[channel][i]);
				if (v->flags & VCO_STEREO)
					signalcount++;
			}
			// adjust for pop in store
			if (SynthObjP->InstrumentValues[channel][i][0] == M_FST)
			{
				FST_valP v = (FST_valP)(SynthObjP->InstrumentValues[channel][i]);
				if (v->type & FST_POP)
					signalcount--;
			}
			// check stack undeflow
			if (signalcount < 0)
			{
				valid = 0;
				Go4kVSTi_ClearInstrumentWorkspace(channel);
				for (int j = i; j < MAX_UNITS; j++)
				{
					char txt[10];
					sprintf(txt, "%d", signalcount);
					SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_SIGNALCOUNT1+j*6), "inv");
				}
				char err[128];
				sprintf(err, "Signal INVALID! Unit %d Stack Underflow %d", i+1, signalcount);
				SetWindowText(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), err);
				break;
			}
			// check stack overflow
			else if (signalcount > 8)
			{
				valid = 0;
				Go4kVSTi_ClearInstrumentWorkspace(channel);
				for (int j = i; j < MAX_UNITS; j++)
				{
					char txt[10];
					sprintf(txt, "%d", signalcount);
					SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_SIGNALCOUNT1+j*6), "inv");
				}
				char err[128];
				sprintf(err, "Signal INVALID! Unit %d Stack Overflow %d", i+1, signalcount);
				SetWindowText(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), err);
				break;
			}
			// TODO: check module internal overflow (like needing additional FP slots internally)
			else 
			{
			}
			// update signal counter
			char txt[10];
			sprintf(txt, "%d", signalcount);
			SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_SIGNALCOUNT1+i*6), txt);
		}
	}
	if (valid)
	{
		if (signalcount == 0)
		{
			SynthObjP->InstrumentSignalValid[channel] = 1;
			SetWindowText(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), "Signal OK");
		}
		else
		{
			Go4kVSTi_ClearInstrumentWorkspace(channel);
			SetWindowText(GetDlgItem(DialogWnd, IDC_ISTACK_VALID), "Signal INVALID! Signal Count != 0 after last unit");
		}
	}
	SynthObjP->HighestSlotIndex[channel] = highestSlotIndex;
	if (channel < 16)
	{
		if (LinkToInstrument[channel] != 16)
		{
			SynthObjP->InstrumentSignalValid[LinkToInstrument[channel]] = SynthObjP->InstrumentSignalValid[channel];
			SynthObjP->HighestSlotIndex[LinkToInstrument[channel]] = SynthObjP->HighestSlotIndex[channel];
		}
	}

	//////////////////////////////////////////////////////////////
	//  global
	//////////////////////////////////////////////////////////////

	// disable for safety first
	SynthObjP->GlobalSignalValid = 0;
	valid = 1;
	signalcount = 0;
	// check stack validity
	SynthObjP->HighestSlotIndex[16] = 63;
	highestSlotIndex = 0;	
	for (int i = 0; i < MAX_UNITS; i++)
	{
		if (SynthObjP->GlobalValues[i][0] != 0)
		{
			if (i >= highestSlotIndex)
				highestSlotIndex = i;
		}
		// check unit precondition
		int precond = UnitPreSignals[SynthObjP->GlobalValues[i][0]];
		// adjust for arithmetic unit (depending on mode)
		if (SynthObjP->GlobalValues[i][0] == M_FOP)
		{
			FOP_valP v = (FOP_valP)(SynthObjP->GlobalValues[i]);
			if (v->flags == FOP_LOADNOTE)
				precond = 0;
			if (v->flags == FOP_POP)
				precond = 1;
			if (v->flags == FOP_PUSH)
				precond = 1;
			if (v->flags == FOP_ADDP2)
				precond = 4;
			if (v->flags == FOP_MULP2)
				precond = 4;
		}
		// adjust for stereo in vcf (we need 2 signals for stereo)
		if (SynthObjP->GlobalValues[i][0] == M_VCF)
		{
			VCF_valP v = (VCF_valP)(SynthObjP->GlobalValues[i]);
			if (v->type & VCF_STEREO)
				precond += 1;
		}
		// adjust for stereo in dst (we need 2 signals for stereo)
		if (SynthObjP->GlobalValues[i][0] == M_DST)
		{
			DST_valP v = (DST_valP)(SynthObjP->GlobalValues[i]);
			if (v->stereo & VCF_STEREO)
				precond += 1;
		}
		if (signalcount < precond)
		{
			valid = 0;
			Go4kVSTi_ClearGlobalWorkspace();
			for (int j = i; j < MAX_UNITS; j++)
			{
				char txt[10];
				sprintf(txt, "%d", signalcount);
				SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_SIGNALCOUNT1+j*6), "inv");
			}
			char err[128];
			sprintf(err, "Signal INVALID! Unit %d needs >= %d signals", i+1, precond);
			SetWindowText(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), err);
			break;
		}
		// precondition ok
		else
		{
			signalcount += UnitPostSignals[SynthObjP->GlobalValues[i][0]];
			// adjust for arithmetic unit (depending on mode)
			if (SynthObjP->GlobalValues[i][0] == M_FOP)
			{
				FOP_valP v = (FOP_valP)(SynthObjP->GlobalValues[i]);
				if (v->flags == FOP_POP)
					signalcount--;
				if (v->flags == FOP_PUSH)
					signalcount++;
				if (v->flags == FOP_ADDP)
					signalcount--;
				if (v->flags == FOP_MULP)
					signalcount--;
				if (v->flags == FOP_ADDP2)
					signalcount-=2;
				if (v->flags == FOP_MULP2)
					signalcount-=2;
				if (v->flags == FOP_LOADNOTE)
					signalcount++;
			}
			// adjust for stereo in vco (2 signals for stereo)
			if (SynthObjP->GlobalValues[i][0] == M_VCO)
			{
				VCO_valP v = (VCO_valP)(SynthObjP->GlobalValues[i]);
				if (v->flags & VCO_STEREO)
					signalcount++;
			}
			// adjust for pop in store
			if (SynthObjP->GlobalValues[i][0] == M_FST)
			{
				FST_valP v = (FST_valP)(SynthObjP->GlobalValues[i]);
				if (v->type & FST_POP)
					signalcount--;
			}
			// check stack undeflow
			if (signalcount < 0)
			{
				valid = 0;
				Go4kVSTi_ClearGlobalWorkspace();
				for (int j = i; j < MAX_UNITS; j++)
				{
					char txt[10];
					sprintf(txt, "%d", signalcount);
					SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_SIGNALCOUNT1+j*6), "inv");
				}
				char err[128];
				sprintf(err, "Signal INVALID! Unit %d Stack Underflow %d", i+1, signalcount);
				SetWindowText(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), err);
				break;
			}
			// check stack overflow
			else if (signalcount > 8)
			{
				valid = 0;
				Go4kVSTi_ClearGlobalWorkspace();
				for (int j = i; j < MAX_UNITS; j++)
				{
					char txt[10];
					sprintf(txt, "%d", signalcount);
					SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_SIGNALCOUNT1+j*6), "inv");
				}
				char err[128];
				sprintf(err, "Signal INVALID! Unit %d Stack Overflow %d", i+1, signalcount);
				SetWindowText(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), err);
				break;
			}
			// TODO: check module internal overflow (like needing additional FP slots internally)
			else 
			{
			}
			// update signal counter
			char txt[10];
			sprintf(txt, "%d", signalcount);
			SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_SIGNALCOUNT1+i*6), txt);
		}
	}
	if (valid)
	{
		if (signalcount == 0)
		{
			SynthObjP->GlobalSignalValid = 1;
			SetWindowText(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), "Signal OK");
		}
		else
		{
			Go4kVSTi_ClearGlobalWorkspace();
			SetWindowText(GetDlgItem(DialogWnd, IDC_GSTACK_VALID), "Signal INVALID! Signal Count != 0 after last unit");
		}
	}
	SynthObjP->HighestSlotIndex[16] = highestSlotIndex;	
}

void UpdateControls(int channel)
{
	int res;
	UpdateSignalCount(channel);
	SetDlgItemText(DialogWnd, IDC_INSTRUMENT_NAME, (LPSTR)&(SynthObjP->InstrumentNames[channel]));
	SendDlgItemMessage(DialogWnd, IDC_INSTRUMENTLINK, CB_SETCURSEL, (WPARAM)LinkToInstrument[SelectedInstrument], (LPARAM)0);
	SendDlgItemMessage(DialogWnd, IDC_POLYPHONY, CB_SETCURSEL, (WPARAM)(SynthObjP->Polyphony-1), (LPARAM)0);
	for (int i = 0; i < MAX_UNITS; i++)
	{
		// set unit text
		char unitname[128];
		GetUnitString(SynthObjP->InstrumentValues[channel][i], unitname);
		SetWindowText(GetDlgItem(TabWnd[T_INSTRUMENT], IDC_ISTACK_UNIT1+i*6), unitname);
		GetUnitString(SynthObjP->GlobalValues[i], unitname);
		SetWindowText(GetDlgItem(TabWnd[T_GLOBAL], IDC_GSTACK_UNIT1+i*6), unitname);
	}
	InterleavedButtonGroupChanged(IDC_ISTACK_UNIT1, MAX_UNITS, IDC_ISTACK_UNIT1+SelectedIUnit*6, res);
	InterleavedButtonGroupChanged(IDC_GSTACK_UNIT1, MAX_UNITS, IDC_GSTACK_UNIT1+SelectedGUnit*6, res);
	UpdateModuleParamWindow(SelectedTab, -1);
}

void UpdateModule(int uid, BYTE* val)
{
	if (uid == M_ENV)
	{
		ENV_valP v = (ENV_valP)val;
		// attack
		InitSlider(ModuleWnd[uid], IDC_ENV_ATTACK, 0, 128, v->attac);
		// decay
		InitSlider(ModuleWnd[uid], IDC_ENV_DECAY, 0, 128, v->decay);
		// sustain
		InitSlider(ModuleWnd[uid], IDC_ENV_SUSTAIN, 0, 128, v->sustain);
		// release
		InitSlider(ModuleWnd[uid], IDC_ENV_RELEASE, 0, 128, v->release);
		// gain
		InitSlider(ModuleWnd[uid], IDC_ENV_GAIN, 0, 128, v->gain);
	}
	else if (uid == M_VCO)
	{
		VCO_valP v = (VCO_valP)val;
		// transpose
		InitSliderCenter(ModuleWnd[uid], IDC_VCO_TRANSPOSE, 0, 128, v->transpose);
		// detune
		InitSliderCenter(ModuleWnd[uid], IDC_VCO_DETUNE, 0, 128, v->detune);
		// phaseofs
		InitSlider(ModuleWnd[uid], IDC_VCO_PHASE, 0, 128, v->phaseofs);
		// color
		InitSliderCenter(ModuleWnd[uid], IDC_VCO_COLOR, 0, 128, v->color);
		// shape
		InitSliderCenter(ModuleWnd[uid], IDC_VCO_SHAPE, 0, 128, v->shape);
		// gain
		InitSlider(ModuleWnd[uid], IDC_VCO_GAIN, 0, 128, v->gain);
		
		// buttons
		DisableButtonGroup(IDC_VCO_SINE, IDC_VCO_GATE, M_VCO);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE_VAL), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE_VAL), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE_VAL), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR_VAL), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE), true);
		EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE_VAL), true);
		EnableButtonGroup(IDC_VCO_GATE1, IDC_VCO_GATE16, M_VCO, false);
		if (v->flags & VCO_SINE)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SINE), false);
		}
		else if (v->flags & VCO_TRISAW)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRISAW), false);
		}
		else if (v->flags & VCO_PULSE)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PULSE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_SHAPE_VAL), false);
		}
		else if (v->flags & VCO_NOISE)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_NOISE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_TRANSPOSE_VAL), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_DETUNE_VAL), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_PHASE_VAL), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR_VAL), false);
		}
		else if (v->flags & VCO_GATE)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_GATE), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_VCO], IDC_VCO_COLOR_VAL), false);
			EnableButtonGroup(IDC_VCO_GATE1, IDC_VCO_GATE16, M_VCO, true);
		}

		// gate bits
		WORD gatebits = ((WORD)v->color << 8) | (WORD)v->gate;
		SetCheckboxGroupBitmask(IDC_VCO_GATE1, IDC_VCO_GATE16, M_VCO, gatebits);
		SendDlgItemMessage(ModuleWnd[M_VCO], IDC_VCO_LFO, BM_SETCHECK, v->flags & VCO_LFO, 0);	
		SendDlgItemMessage(ModuleWnd[M_VCO], IDC_VCO_STEREO, BM_SETCHECK, v->flags & VCO_STEREO, 0);
	}
	else if (uid == M_VCF)
	{
		VCF_valP v = (VCF_valP)val;
		// frequency
		InitSlider(ModuleWnd[uid], IDC_VCF_FREQUENCY, 0, 128, v->freq);
		// resonance
		InitSlider(ModuleWnd[uid], IDC_VCF_RESONANCE, 0, 128, v->res);

		// buttons
		DisableButtonGroup(IDC_VCF_LOW, IDC_VCF_ALL, M_VCF);

		int mode = v->type & ~VCF_STEREO;
		if (mode == VCF_LOWPASS)
			EnableWindow(GetDlgItem(ModuleWnd[M_VCF], IDC_VCF_LOW), false);
		else if (mode == VCF_HIGHPASS)
			EnableWindow(GetDlgItem(ModuleWnd[M_VCF], IDC_VCF_HIGH), false);
		else if (mode == VCF_BANDPASS)
			EnableWindow(GetDlgItem(ModuleWnd[M_VCF], IDC_VCF_BAND), false);
		else if (mode == VCF_BANDSTOP)
			EnableWindow(GetDlgItem(ModuleWnd[M_VCF], IDC_VCF_NOTCH), false);
		else if (mode == VCF_PEAK)
			EnableWindow(GetDlgItem(ModuleWnd[M_VCF], IDC_VCF_PEAK), false);
		else if (mode == VCF_ALLPASS)
			EnableWindow(GetDlgItem(ModuleWnd[M_VCF], IDC_VCF_ALL), false);

		SendDlgItemMessage(ModuleWnd[M_VCF], IDC_VCF_STEREO, BM_SETCHECK, v->type & VCF_STEREO, 0);
	}
	else if (uid == M_DST)
	{
		DST_valP v = (DST_valP)val;
		// drive
		InitSliderCenter(ModuleWnd[uid], IDC_DST_DRIVE, 0, 128, v->drive);
		// snhfreq
		InitSlider(ModuleWnd[uid], IDC_DST_SNH, 0, 128, v->snhfreq);
		SendDlgItemMessage(ModuleWnd[M_DST], IDC_DST_STEREO, BM_SETCHECK, v->stereo & VCF_STEREO, 0);
	}
	else if (uid == M_DLL)
	{
		DLL_valP v = (DLL_valP)val;
		// pregain
		InitSlider(ModuleWnd[uid], IDC_DLL_PREGAIN, 0, 128, v->pregain);
		// dry
		InitSlider(ModuleWnd[uid], IDC_DLL_DRY, 0, 128, v->dry);
		// feedback
		InitSlider(ModuleWnd[uid], IDC_DLL_FEEDBACK, 0, 128, v->feedback);
		// frequency
		InitSlider(ModuleWnd[uid], IDC_DLL_FREQUENCY, 0, 128, v->freq);
		// depth
		InitSlider(ModuleWnd[uid], IDC_DLL_DEPTH, 0, 128, v->depth);
		// damp
		InitSlider(ModuleWnd[uid], IDC_DLL_DAMP, 0, 128, v->damp);
		// delay
		InitSliderNoGUI(ModuleWnd[uid], IDC_DLL_DTIME, 0, 128, v->guidelay);
		
		// delay/reverb button
		DisableButtonGroup(IDC_DLL_DELAY, IDC_DLL_REVERB, M_DLL);
		if (v->reverb)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_REVERB), false);
			// enable reverb gui elements
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_LEFTREVERB), SW_SHOW);
			// disable delay gui elements
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOSYNC), SW_HIDE);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_BPMSYNC), SW_HIDE);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOTESYNC), SW_HIDE);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), SW_HIDE);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), SW_HIDE);
		}
		else
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DELAY), false);	
			// disable reverb gui elements
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_LEFTREVERB), SW_HIDE);
			// enable delay gui elements
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOSYNC), SW_SHOW);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_BPMSYNC), SW_SHOW);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOTESYNC), SW_SHOW);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), SW_SHOW);
			ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), SW_SHOW);
		}
		DisableButtonGroup(IDC_DLL_NOSYNC, IDC_DLL_NOTESYNC, M_DLL);
		if (v->synctype == 0)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), true);
		}
		else if (v->synctype == 1)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), true);
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), true);
		}
		else if (v->synctype == 2)
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME), false);
			EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_DTIME_VAL), false);
		}
		EnableWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_DLL_NOSYNC+v->synctype), false);	
		SendDlgItemMessage(ModuleWnd[M_DLL], IDC_DLL_LEFTREVERB, BM_SETCHECK, v->leftreverb, 0);
		UpdateDelayTimes(v);
	}
	else if (uid == M_FOP)
	{
		FOP_valP v = (FOP_valP)val;
		DisableButtonGroup(IDC_FOP_POP, IDC_FOP_MULP2, M_FOP);
		EnableWindow(GetDlgItem(ModuleWnd[M_FOP], IDC_FOP_POP + v->flags-FOP_POP), false);
	}
	else if (uid == M_FST)
	{
		FST_valP v = (FST_valP)val;
		// gain
		InitSliderCenter2(ModuleWnd[uid], IDC_FST_AMOUNT, 0, 128, v->amount);

		// set instrument combobox
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"Local");
		for (int i = 0; i < MAX_INSTRUMENTS; i++)
		{
			char instname[128];
			sprintf(instname, "%d: %s", i+1, SynthObjP->InstrumentNames[i]);
			SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)instname);
		}
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"Global");
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_INSTRUMENT, CB_SETCURSEL, (WPARAM)(v->dest_stack+1), (LPARAM)0);
		
		// set unit combobox
		BYTE *units;
		if (SelectedTab == T_INSTRUMENT)
		{
			if (v->dest_stack == -1)
				units = SynthObjP->InstrumentValues[SelectedInstrument][0];
			else if (v->dest_stack == MAX_INSTRUMENTS)
				units = SynthObjP->GlobalValues[0];
			else
				units = SynthObjP->InstrumentValues[v->dest_stack][0];
		}
		else
		{
			if (v->dest_stack == -1)
				units = SynthObjP->GlobalValues[0];
			else if (v->dest_stack == MAX_INSTRUMENTS)
				units = SynthObjP->GlobalValues[0];
			else
				units = SynthObjP->InstrumentValues[v->dest_stack][0];
		}

		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"- Nothing Selected -");
		for (int i = 0; i < MAX_UNITS; i++)
		{
			char unitname[128], unitname2[128];
			sprintf(unitname, "%d: %s", i+1, GetUnitString(&units[i*MAX_UNIT_SLOTS], unitname2));
			SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_ADDSTRING, (WPARAM)0, (LPARAM)unitname);
		}
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_UNIT, CB_SETCURSEL, (WPARAM)(v->dest_unit+1), (LPARAM)0);

		// set slot combobox
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_RESETCONTENT, (WPARAM)0, (LPARAM)0);
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_ADDSTRING, (WPARAM)0, (LPARAM)"- Nothing Selected -");
		if (v->dest_unit == -1)
		{
			SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_SETCURSEL, (WPARAM)0, (LPARAM)0);
			EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT), false);
		}
		else
		{
			EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT), true);
			DWORD unitid = units[v->dest_unit*MAX_UNIT_SLOTS];			
			for (int i = 0; i < 8; i++)
			{
				SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_ADDSTRING, (WPARAM)0, (LPARAM)UnitModulationTargetNames[unitid][i]);
			}
			SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_DESTINATION_SLOT, CB_SETCURSEL, (WPARAM)(v->dest_slot+1), (LPARAM)0);
		}
		// buttons
		DisableButtonGroup(IDC_FST_SET, IDC_FST_MUL, M_FST);
		if (v->type & FST_ADD)
			EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_ADD), false);
		else if (v->type & FST_MUL)
			EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_MUL), false);
		else
			EnableWindow(GetDlgItem(ModuleWnd[M_FST], IDC_FST_SET), false);
		SendDlgItemMessage(ModuleWnd[M_FST], IDC_FST_POP, BM_SETCHECK, v->type & FST_POP, 0);
	}
	else if (uid == M_PAN)
	{
		PAN_valP v = (PAN_valP)val;
		// panning
		InitSliderCenter(ModuleWnd[uid], IDC_PAN_PANNING, 0, 128, v->panning);
	}
	else if (uid == M_OUT)
	{
		OUT_valP v = (OUT_valP)val;
		// gain
		InitSlider(ModuleWnd[uid], IDC_OUT_GAIN, 0, 128, v->gain);
		// auxsend
		InitSlider(ModuleWnd[uid], IDC_OUT_AUXSEND, 0, 128, v->auxsend);
	}
	else if (uid == M_ACC)
	{
		ACC_valP v = (ACC_valP)val;
		DisableButtonGroup(IDC_ACC_OUT, IDC_ACC_AUX, M_ACC);
		if (v->flags == ACC_OUT)
			EnableWindow(GetDlgItem(ModuleWnd[M_ACC], IDC_ACC_OUT), false);
		else
			EnableWindow(GetDlgItem(ModuleWnd[M_ACC], IDC_ACC_AUX), false);
	}
	else if (uid == M_FLD)
	{
		FLD_valP v = (FLD_valP)val;
		// panning
		InitSliderCenter(ModuleWnd[uid], IDC_FLD_VALUE, 0, 128, v->value);
	}
	else if (uid == M_GLITCH)
	{
		GLITCH_valP v = (GLITCH_valP)val;
		// active
		InitSlider(ModuleWnd[uid], IDC_GLITCH_ACTIVE, 0, 128, v->active);
		// dry
		InitSlider(ModuleWnd[uid], IDC_GLITCH_DRY, 0, 128, v->dry);
		// delta size
		InitSlider(ModuleWnd[uid], IDC_GLITCH_DSIZE, 0, 128, v->dsize);
		// delta pitch
		InitSlider(ModuleWnd[uid], IDC_GLITCH_DPITCH, 0, 128, v->dpitch);
		// delay
		InitSliderNoGUI(ModuleWnd[uid], IDC_GLITCH_DTIME, 0, 128, v->guidelay);
		
		ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_GLITCH_DTIME), SW_SHOW);
		ShowWindow(GetDlgItem(ModuleWnd[M_DLL], IDC_GLITCH_DTIME_VAL), SW_SHOW);

		UpdateDelayTimes(v);
	}
}

void UpdateModuleParamWindow(int tab, int unit)
{
	if (tab == T_INSTRUMENT)
	{
		// show unit page
		if (unit == -1)
			unit = SelectedIUnit;
		for (int i = 0; i < NUM_MODULES; i++)
		{
			if (i == SynthObjP->InstrumentValues[SelectedInstrument][unit][0])
				ShowWindow(ModuleWnd[M_NONE+i], SW_SHOW);
			else
				ShowWindow(ModuleWnd[M_NONE+i], SW_HIDE);
		}
		// set unit sliders and buttons
		UnitID uid = (UnitID)(SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit][0]);
		UpdateModule(uid, SynthObjP->InstrumentValues[SelectedInstrument][SelectedIUnit]);
	}
	else if (tab == T_GLOBAL)
	{
		// show unit page
		if (unit == -1)
			unit = SelectedGUnit;
		for (int i = 0; i < NUM_MODULES; i++)
		{
			if (i == SynthObjP->GlobalValues[unit][0])
				ShowWindow(ModuleWnd[M_NONE+i], SW_SHOW);
			else
				ShowWindow(ModuleWnd[M_NONE+i], SW_HIDE);
		}
		// set unit sliders and buttons
		UnitID uid = (UnitID)(SynthObjP->GlobalValues[SelectedGUnit][0]);
		UpdateModule(uid, SynthObjP->GlobalValues[SelectedGUnit]);
	}
}

void UpdateVoiceDisplay(int i)
{
	for (int i = 0; i < MAX_INSTRUMENTS; i++)
	{
		int count = 0;
		for (int p = 0; p < SynthObjP->Polyphony; p++)
		{
			if (SynthObjP->InstrumentWork[i*MAX_POLYPHONY+p].note != 0)
				count++;
		}
		char text[10];
		if (SynthObjP->InstrumentSignalValid[i])
		{
			sprintf(text, "%d", count);
			SetWindowText(GetDlgItem(DialogWnd, IDC_VOICECOUNT1+i), text);
		}
		else
		{
			sprintf(text, "inv", count);
			SetWindowText(GetDlgItem(DialogWnd, IDC_VOICECOUNT1+i), text);
		}
	}
}

void GetStreamFileName()
{
#ifdef EXPORT_OBJECT_FILE
	
	char lpstrFile[256] = {'4','k','l','a','n','g','.','o','b','j',0};
	char lpstrFilter[256] = {'4','k','l','a','n','g',' ','O','b','j','e','c','t', 0 ,'*','.','o','b','j', 0, 0};
	char lpstrExtension[256] = { 'o','b','j', 0 };
	char lpstrFileTitle[4096];
	char lpstrDirectory[4096];
	GetCurrentDirectory(4096, lpstrDirectory);

	// no windows object?
	int objformat = SendDlgItemMessage(DialogWnd, IDC_OBJFORMAT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
	if (objformat != 0)
	{
		lpstrFile[8] = 0; // 4klang.o
		lpstrFilter[17] = lpstrFilter[18] = 0;
		lpstrExtension[1] = 0;
	}

	OPENFILENAME ofn =
	{
		sizeof(OPENFILENAME),
		DialogWnd,
		GetModuleHandle(NULL),
		lpstrFilter,
		NULL,
		0,
		1,
		lpstrFile,
		256,
		NULL,
		0,
		lpstrDirectory,
		NULL,
		OFN_OVERWRITEPROMPT,
		0,
		0,
		lpstrExtension,
		0,
		NULL,
		NULL		
	};
#else
	char lpstrFilter[256] =
	{'4','k','l','a','n','g',' ','I','n','c','l','u','d','e', 0 ,
	'*','.','i','n','c', 0, 
	0};
#ifdef _4KLANG2
	char lpstrFile[4096] = "4klang2.inc";
#else
	char lpstrFile[4096] = "4klang.inc";
#endif
	char lpstrDirectory[4096];
	GetCurrentDirectory(4096, lpstrDirectory);

	OPENFILENAME ofn =
	{
		sizeof(OPENFILENAME),
		DialogWnd,
		GetModuleHandle(NULL),
		lpstrFilter,
		NULL,
		0,
		1,
		lpstrFile,
		256,
		NULL,
		0,
		lpstrDirectory,
		NULL,
		OFN_OVERWRITEPROMPT,
		0,
		0,
		"inc",
		0,
		NULL,
		NULL		
	};
#endif
	if (GetSaveFileName(&ofn))
	{
		int useenvlevels = SendDlgItemMessage(DialogWnd, IDC_ENVLEVELS, BM_GETCHECK, 0, 0);
		int useenotevalues = SendDlgItemMessage(DialogWnd, IDC_NOTEVALUES, BM_GETCHECK, 0, 0);
		int clipoutput = SendDlgItemMessage(DialogWnd, IDC_CLIPOUTPUT, BM_GETCHECK, 0, 0);
		int undenormalize = SendDlgItemMessage(DialogWnd, IDC_UNDENORMALIZE, BM_GETCHECK, 0, 0);
		int objformat = SendDlgItemMessage(DialogWnd, IDC_OBJFORMAT, CB_GETCURSEL, (WPARAM)0, (LPARAM)0);
		int output16 = SendDlgItemMessage(DialogWnd, IDC_16BIT, BM_GETCHECK, 0, 0);
		Go4kVSTi_SaveByteStream(hInstance, ofn.lpstrFile, useenvlevels, useenotevalues, clipoutput, undenormalize, objformat, output16);
	}
}

