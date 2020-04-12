#include <stdio.h>
#include <windows.h>
#ifndef __Go4kVSTi__
#include "Go4kVSTi.h"
#endif
#include "Go4kVSTiCore.h"
//-----------------------------------------------------------------------------------------
// Go4kVSTi
//-----------------------------------------------------------------------------------------

//-----------------------------------------------------------------------------------------
Go4kVSTi::Go4kVSTi (audioMasterCallback audioMaster) : AudioEffectX (audioMaster, 0, 0)
{
	m_chunkBuffer = 0;
	if (audioMaster)
	{
		setNumInputs (0);				// no inputs
		setNumOutputs (2);				// 2 outputs, stereo
		canProcessReplacing ();
		hasVu (false);
		hasClip (false);
		isSynth ();
		programsAreChunks (true);
#ifdef _8KLANG
		setUniqueID ('8klg');
#else
		setUniqueID ('4klg');
#endif
	}
	initProcess ();
	suspend ();
}

//-----------------------------------------------------------------------------------------
Go4kVSTi::~Go4kVSTi ()
{
	delete[] m_chunkBuffer;
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::setProgram (long program)
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::setProgramName (char *name)
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::getProgramName (char *name)
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::getParameterLabel (long index, char *label)
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::getParameterDisplay (long index, char *text)
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::getParameterName (long index, char *label)
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::setParameter (long index, float value)
{
}

//-----------------------------------------------------------------------------------------
float Go4kVSTi::getParameter (long index)
{
	return 0;
}

//-----------------------------------------------------------------------------------------
bool Go4kVSTi::getOutputProperties (long index, VstPinProperties* properties)
{
	if (index < 2)
	{
		sprintf (properties->label, "Vstx %1d", index + 1);
		properties->flags = kVstPinIsActive;
		properties->flags |= kVstPinIsStereo;	// test, make channel 1+2 stereo
		return true;
	}
	return false;
}

//-----------------------------------------------------------------------------------------
bool Go4kVSTi::getProgramNameIndexed (long category, long index, char* text)
{
	return false;
}

//-----------------------------------------------------------------------------------------
bool Go4kVSTi::copyProgram (long destination)
{
	return false;
}

//-----------------------------------------------------------------------------------------
bool Go4kVSTi::getEffectName (char* name)
{
#ifdef _8KLANG
	strcpy (name, "8klang");
#else
	strcpy (name, "4klang");
#endif
	return true;
}

//-----------------------------------------------------------------------------------------
bool Go4kVSTi::getVendorString (char* text)
{
	strcpy (text, "Alcatraz");
	return true;
}

//-----------------------------------------------------------------------------------------
bool Go4kVSTi::getProductString (char* text)
{
#ifdef _8KLANG
	strcpy (text, "8klang");
#else
	strcpy (text, "4klang");
#endif
	return true;
}

//-----------------------------------------------------------------------------------------
long Go4kVSTi::canDo (char* text)
{
	if (!strcmp (text, "receiveVstEvents"))
		return 1;
	if (!strcmp (text, "receiveVstMidiEvent"))
		return 1;
	if (!strcmp (text, "receiveVstTimeInfo"))
		return 1;
	return -1;	// explicitly can't do; 0 => don't know
}

//-----------------------------------------------------------------------------------------
long Go4kVSTi::getChunk(void** data, bool isPreset)
{
	// serialize patch data into file, then load it back
	char path[MAX_PATH];
	char filename[MAX_PATH];
	GetTempPath(MAX_PATH, path);
	GetTempFileName(path, "4klang_", 0, filename);
	Go4kVSTi_SavePatch(filename);

	HANDLE h = CreateFile(filename, GENERIC_READ, 0, 0, OPEN_EXISTING, 0, 0);
	if (h == INVALID_HANDLE_VALUE)
		return 0;

	DWORD dummy;
	DWORD size = GetFileSize(h, &dummy);
	delete[] m_chunkBuffer;
	m_chunkBuffer = new unsigned char[size];
	ReadFile(h, m_chunkBuffer, size, &dummy, 0);
	CloseHandle(h);
	DeleteFile(filename);

	if (dummy == size)
	{
		*data = m_chunkBuffer;
		return size;
	}
	return 0;
}

//-----------------------------------------------------------------------------------------
long Go4kVSTi::setChunk(void* data, long byteSize, bool isPreset)
{
	if (!data || !byteSize) return 0;

	// write chunk into file, then deserialize as patch data
	char path[MAX_PATH];
	char filename[MAX_PATH];
	GetTempPath(MAX_PATH, path);
	GetTempFileName(path, "4klang_", 0, filename);

	HANDLE h = CreateFile(filename, GENERIC_WRITE, 0, 0, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, 0);
	if (h == INVALID_HANDLE_VALUE)
	{
		DWORD err = GetLastError();
		return 0;
	}
	DWORD dummy;
	WriteFile(h, data, byteSize, &dummy, 0);
	CloseHandle(h);
	
	if (dummy == byteSize)
	{
		Go4kVSTi_LoadPatch(filename);
	}
	DeleteFile(filename);
	return 0;
}
