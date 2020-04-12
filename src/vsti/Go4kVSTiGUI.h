#include <windows.h>

#define ButtonGroupChanged(btn_firstid, btn_lastid, test_id, seltab, result_index)	\
{																					\
	result_index = 0;																\
	if (LOWORD(test_id) >= btn_firstid &&											\
		LOWORD(test_id) <= btn_lastid)												\
	{																				\
		for (WORD bgci = btn_firstid; bgci <= btn_lastid; bgci++)					\
		{																			\
			if (LOWORD(test_id) == bgci)											\
			{																		\
				EnableWindow(GetDlgItem(ModuleWnd[seltab], bgci), false);			\
				result_index = bgci;												\
			}																		\
			else																	\
			{																		\
				EnableWindow(GetDlgItem(ModuleWnd[seltab], bgci), true);			\
			}																		\
		}																			\
	}																				\
}							

#define EnableButtonGroup(btn_firstid, btn_lastid, seltab, btn_show)				\
{																					\
	for (WORD bgci = btn_firstid; bgci <= btn_lastid; bgci++)						\
	{																				\
		EnableWindow(GetDlgItem(ModuleWnd[seltab], bgci), btn_show);				\
	}																				\
}		

#define SetCheckboxGroupBitmask(btn_firstid, btn_lastid, seltab, input_bits)		\
{																					\
	WORD curbitsel = 0x0001;														\
	for (WORD bgci = btn_firstid; bgci <= btn_lastid; bgci++,curbitsel*=2)			\
	{																				\
		SendDlgItemMessage(ModuleWnd[seltab], bgci, BM_SETCHECK, input_bits & curbitsel, 0);	\
	}																				\
}

#define GetCheckboxGroupBitmask(btn_firstid, btn_lastid, test_id, seltab, result_bits, result_index)	\
{																					\
	result_index = 0;																\
	result_bits = 0;																\
	WORD curbitsel = 0x0001;														\
	if (LOWORD(test_id) >= btn_firstid &&											\
		LOWORD(test_id) <= btn_lastid)												\
	{																				\
		for (WORD bgci = btn_firstid; bgci <= btn_lastid; bgci++,curbitsel*=2)		\
		{																			\
			if (LOWORD(test_id) == bgci)											\
			{																		\
				result_index = bgci;												\
			}																		\
			if (SendDlgItemMessage(ModuleWnd[seltab], bgci, BM_GETCHECK, 0, 0)==BST_CHECKED)	\
			{																		\
				result_bits |= curbitsel;											\
			}																		\
			else																	\
			{																		\
				result_bits &= ~curbitsel;											\
			}																		\
		}																			\
	}																				\
}

#define InitSlider(tab_id, slider_id, slider_low, slider_hi, slider_pos)						\
{																								\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETRANGE, true, MAKELONG(slider_low, slider_hi)); \
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPOS, true, slider_pos);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETLINESIZE, 0, 1);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPAGESIZE, 0, 1);						\
	sprintf(SliderValTxt, "%d", slider_pos);											\
	SetWindowText(GetDlgItem(tab_id, slider_id ## _VAL), SliderValTxt);					\
}

#define InitSliderCenter(tab_id, slider_id, slider_low, slider_hi, slider_pos)						\
{																								\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETRANGE, true, MAKELONG(slider_low, slider_hi)); \
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPOS, true, slider_pos);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETLINESIZE, 0, 1);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPAGESIZE, 0, 1);						\
	sprintf(SliderValTxt, "%d", slider_pos-64);											\
	SetWindowText(GetDlgItem(tab_id, slider_id ## _VAL), SliderValTxt);					\
}

#define InitSliderCenter2(tab_id, slider_id, slider_low, slider_hi, slider_pos)						\
{																								\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETRANGE, true, MAKELONG(slider_low, slider_hi)); \
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPOS, true, slider_pos);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETLINESIZE, 0, 1);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPAGESIZE, 0, 1);						\
	sprintf(SliderValTxt, "%d", (slider_pos-64)*2);											\
	SetWindowText(GetDlgItem(tab_id, slider_id ## _VAL), SliderValTxt);					\
}

#define InitSliderNoGUI(tab_id, slider_id, slider_low, slider_hi, slider_pos)						\
{																								\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETRANGE, true, MAKELONG(slider_low, slider_hi)); \
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPOS, true, slider_pos);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETLINESIZE, 0, 1);						\
	SendDlgItemMessage(tab_id, slider_id, TBM_SETPAGESIZE, 0, 1);						\
}

#define DisableButtonGroup(btn_firstid, btn_lastid, seltab)					\
{																			\
	for (WORD bgci = btn_firstid; bgci <= btn_lastid; bgci++)				\
	{																		\
		EnableWindow(GetDlgItem(ModuleWnd[seltab], bgci), true);			\
	}																		\
}			

#define InterleavedButtonGroupChanged(btn_firstid, btn_count, test_id, result_index)\
{																					\
	result_index = -1;																\
	if (((LOWORD(test_id)-btn_firstid) % 6) == 0)									\
	{																				\
		for (WORD bgci = 0; bgci < btn_count; bgci++)								\
		{																			\
			WORD ibtn = btn_firstid+bgci*6;											\
			if (LOWORD(test_id) == ibtn)											\
			{																		\
				EnableWindow(GetDlgItem(TabWnd[SelectedTab], ibtn), false);			\
				result_index = bgci;												\
			}																		\
			else																	\
			{																		\
				EnableWindow(GetDlgItem(TabWnd[SelectedTab], ibtn), true);			\
			}																		\
		}																			\
	}																				\
}							

#define DisableInterleavedButtonGroup(btn_firstid, btn_count, seltab)		\
{																			\
	for (WORD bgci = 0; bgci < btn_count; bgci++)							\
	{																		\
		EnableWindow(GetDlgItem(TabWnd[seltab], btn_firstid+bgci*6), true);	\
	}																		\
}			

#define UpdateSliderValue(usv_id, usv_v)													\
{																							\
	usv_v = SendMessage(GetDlgItem(ModuleWnd[uid], usv_id), TBM_GETPOS, 0, 0);				\
	sprintf(SliderValTxt, "%d", usv_v);														\
	SetWindowText(GetDlgItem(ModuleWnd[uid], usv_id ## _VAL), SliderValTxt);					\
}		

#define UpdateSliderValueCenter(usv_id, usv_v)													\
{																							\
	usv_v = SendMessage(GetDlgItem(ModuleWnd[uid], usv_id), TBM_GETPOS, 0, 0);				\
	sprintf(SliderValTxt, "%d", usv_v-64);														\
	SetWindowText(GetDlgItem(ModuleWnd[uid], usv_id ## _VAL), SliderValTxt);					\
}

#define UpdateSliderValueCenter2(usv_id, usv_v)													\
{																							\
	usv_v = SendMessage(GetDlgItem(ModuleWnd[uid], usv_id), TBM_GETPOS, 0, 0);				\
	sprintf(SliderValTxt, "%d", (usv_v-64)*2);														\
	SetWindowText(GetDlgItem(ModuleWnd[uid], usv_id ## _VAL), SliderValTxt);					\
}

void Go4kVSTiGUI_Create(HINSTANCE hInst);
void Go4kVSTiGUI_Show(int showCommand);
void Go4kVSTiGUI_Destroy();

bool InitTabDlg();

bool ButtonPressed(WPARAM id, LPARAM lParam);
bool ScrollbarChanged(HWND hwndDlg, WPARAM wParam, LPARAM lParam);
bool StackButtonPressed(WPARAM id);
void TabChanged(int index);

void UpdateSignalCount(int channel);
void UpdateControls(int channel);
void UpdateModuleParamWindow(int tab, int unit);
void UpdateVoiceDisplay(int i);

void GetStreamFileName();



