#include <windows.h>
#include <stddef.h>
#include "Go4kVSTi.h"
#include "Go4kVSTiGUI.h"

static AudioEffect *effect = 0;
bool oome = false;

//------------------------------------------------------------------------
// prototype of the export function main
int main (audioMasterCallback audioMaster);

int main (audioMasterCallback audioMaster)
{
	// get vst version
	if(!audioMaster (0, audioMasterVersion, 0, 0, 0, 0))
		return 0;  // old version

	effect = new Go4kVSTi (audioMaster);
	if (!effect)
		return 0;
	if (oome)
	{
		delete effect;
		return 0;
	}
	return (int)effect->getAeffect();
}

void* hInstance;
BOOL WINAPI DllMain (HINSTANCE hInst, DWORD dwReason, LPVOID lpvReserved)
{
	switch (dwReason)
	{
		// wird aufgerufen, wenn zum ersten mal das plugin aktiviert wird
		// also hier init für den synth
		case DLL_PROCESS_ATTACH:
            //MessageBox(NULL, "DLL_PROCESS_ATTACH", "DllMain", MB_OK);
			Go4kVSTiGUI_Create(hInst);
			Go4kVSTiGUI_Show(SW_SHOW);
			break;
		// wird aufgerufen, wenn das plugin nicht mehr verwendet wird
		// entweder bei entfernen des letzten, schliessen des songs oder
		// des programms. Also hier Deinit und zerstörung des Synths
		case DLL_PROCESS_DETACH:
			//MessageBox(NULL, "DLL_PROCESS_DETACH", "DllMain", MB_OK);
			Go4kVSTiGUI_Destroy();
			effect = 0;			
			break;
		// die beiden brauchts wohl nicht
		case DLL_THREAD_ATTACH:
			//MessageBox(NULL, "DLL_THREAD_ATTACH", "DllMain", MB_OK);
			break;
		case DLL_THREAD_DETACH:
			//MessageBox(NULL, "DLL_THREAD_DETACH", "DllMain", MB_OK);
			break;
	}
	hInstance = hInst;
	return 1;
}
