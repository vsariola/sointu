#ifndef __audioeffectx__
#include "audioeffectx.h"
#endif

// *** steinberg developers: this is a public file, *do not edit!*

//-------------------------------------------------------------------------------------------------------
// VST Plug-Ins SDK
// version 2.0 extension
// (c)1999 Steinberg Soft+Hardware GmbH
//
// you should not have to edit this file
// use override methods instead, as suggested in the class declaration (audioeffectx.h)
//-------------------------------------------------------------------------------------------------------

//---------------------------------------------------------------------------------------------
// 'canDo' strings. note other 'canDos' can be evaluated by calling the according
// function, for instance if getSampleRate returns 0, you
// will certainly want to assume that this selector is not supported.
//---------------------------------------------------------------------------------------------

const char* hostCanDos [] =
{
	"sendVstEvents",
	"sendVstMidiEvent",
	"sendVstTimeInfo",
	"receiveVstEvents",
	"receiveVstMidiEvent",
	"receiveVstTimeInfo",
	
	"reportConnectionChanges",
	"acceptIOChanges",
	"sizeWindow",

	"asyncProcessing",
	"offline",
	"supplyIdle",
	"supportShell"		// 'shell' handling via uniqueID as suggested by Waves
};

const char* plugCanDos [] =
{
	"sendVstEvents",
	"sendVstMidiEvent",
	"sendVstTimeInfo",
	"receiveVstEvents",
	"receiveVstMidiEvent",
	"receiveVstTimeInfo",
	"offline",
	"plugAsChannelInsert",
	"plugAsSend",
	"mixDryWet",
	"noRealTime",
	"multipass",
	"metapass",
	"1in1out",
	"1in2out",
	"2in1out",
	"2in2out",
	"2in4out",
	"4in2out",
	"4in4out",
	"4in8out",	// 4:2 matrix to surround bus
	"8in4out",	// surround bus to 4:2 matrix
	"8in8out"
};

//-------------------------------------------------------------------------------------------------------
//-------------------------------------------------------------------------------------------------------
// AudioEffectX extends AudioEffect with the new features. so you should derive
// your plug from AudioEffectX
//-------------------------------------------------------------------------------------------------------
//-------------------------------------------------------------------------------------------------------

//-------------------------------------------------------------------------------------------------------
// VstEvents + VstTimeInfo
//-------------------------------------------------------------------------------------------------------

//-------------------------------------------------------------------------------------------------------
AudioEffectX::AudioEffectX (audioMasterCallback audioMaster, long numPrograms, long numParams)
	: AudioEffect (audioMaster, numPrograms, numParams)
{
}

//-------------------------------------------------------------------------------------------------------
AudioEffectX::~AudioEffectX ()
{
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::dispatcher (long opCode, long index, long value, void *ptr, float opt)
{
	long v = 0;
	switch(opCode)
	{
		// VstEvents
		case effProcessEvents:
			v = processEvents ((VstEvents*)ptr);
			break;

		// parameters and programs
		case effCanBeAutomated:
			v = canParameterBeAutomated (index) ? 1 : 0;
			break;
		case effString2Parameter:
			v = string2parameter (index, (char*)ptr) ? 1 : 0;
			break;

		case effGetNumProgramCategories:
			v = getNumCategories ();
			break;
		case effGetProgramNameIndexed:
			v = getProgramNameIndexed (value, index, (char*)ptr) ? 1 : 0;
			break;
		case effCopyProgram:
			v = copyProgram (index) ? 1 : 0;
			break;

		// connections, configuration
		case effConnectInput:
			inputConnected (index, value ? true : false);
			v = 1;
			break;	
		case effConnectOutput:
			outputConnected (index, value ? true : false);
			v = 1;
			break;	
		case effGetInputProperties:
			v = getInputProperties (index, (VstPinProperties*)ptr) ? 1 : 0;
			break;
		case effGetOutputProperties:
			v = getOutputProperties (index, (VstPinProperties*)ptr) ? 1 : 0;
			break;
		case effGetPlugCategory:
			v = (long)getPlugCategory ();
			break;

		// realtime
		case effGetCurrentPosition:
			v = reportCurrentPosition ();
			break;
			
		case effGetDestinationBuffer:
			v = (long)reportDestinationBuffer ();
			break;

		// offline
		case effOfflineNotify:
			v = offlineNotify ((VstAudioFile*)ptr, value, index != 0);
			break;
		case effOfflinePrepare:
			v = offlinePrepare ((VstOfflineTask*)ptr, value);
			break;
		case effOfflineRun:
			v = offlineRun ((VstOfflineTask*)ptr, value);
			break;

		// other
		case effSetSpeakerArrangement:
			v = setSpeakerArrangement ((VstSpeakerArrangement*)value, (VstSpeakerArrangement*)ptr) ? 1 : 0;
			break;
		case effProcessVarIo:
			v = processVariableIo ((VstVariableIo*)ptr) ? 1 : 0;
			break;
		case effSetBlockSizeAndSampleRate:
			setBlockSizeAndSampleRate (value, opt);
			v = 1;
			break;
		case effSetBypass:
			v = setBypass (value ? true : false) ? 1 : 0;
			break;
		case effGetEffectName:
			v = getEffectName ((char *)ptr) ? 1 : 0;
			break;
		case effGetErrorText:
			v = getErrorText ((char *)ptr) ? 1 : 0;
			break;
		case effGetVendorString:
			v = getVendorString ((char *)ptr) ? 1 : 0;
			break;
		case effGetProductString:
			v = getProductString ((char *)ptr) ? 1 : 0;
			break;
		case effGetVendorVersion:
			v = getVendorVersion ();
			break;
		case effVendorSpecific:
			v = vendorSpecific (index, value, ptr, opt);
			break;
		case effCanDo:
			v = canDo ((char*)ptr);
			break;
		case effGetIcon:
			v = (long)getIcon ();
			break;
		case effSetViewPosition:
			v = setViewPosition (index, value) ? 1 : 0;
			break;		
		case effGetTailSize:
			v = getGetTailSize ();
			break;
		case effIdle:
			v = fxIdle ();
			break;

		case effGetParameterProperties:
			v = getParameterProperties (index, (VstParameterProperties*)ptr) ? 1 : 0;
			break;

		case effKeysRequired:
			v = (keysRequired () ? 0 : 1);	// reversed to keep v1 compatibility
			break;
		case effGetVstVersion:
			v = getVstVersion ();
			break;

		// version 1.0 or unknown
		default:
			v = AudioEffect::dispatcher (opCode, index, value, ptr, opt);
	}
	return v;
}

//-------------------------------------------------------------------------------------------------------
void AudioEffectX::wantEvents (long filter)
{
	if (audioMaster)
		audioMaster (&cEffect, audioMasterWantMidi, 0, filter, 0, 0);
	
}

//-------------------------------------------------------------------------------------------------------
VstTimeInfo* AudioEffectX::getTimeInfo (long filter)
{
	if (audioMaster)
		return (VstTimeInfo*) audioMaster (&cEffect, audioMasterGetTime, 0, filter, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::tempoAt (long pos)
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterTempoAt, 0, pos, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::sendVstEventsToHost (VstEvents* events)
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterProcessEvents, 0, 0, events, 0) == 1;
	return 0;
}

//-------------------------------------------------------------------------------------------------------
// parameters
//-------------------------------------------------------------------------------------------------------

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getNumAutomatableParameters ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetNumAutomatableParameters, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getParameterQuantization ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetParameterQuantization, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
// configuration
//-------------------------------------------------------------------------------------------------------

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::ioChanged ()
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterIOChanged, 0, 0, 0, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::needIdle ()
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterNeedIdle, 0, 0, 0, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::sizeWindow (long width, long height)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterSizeWindow, width, height, 0, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
double AudioEffectX::updateSampleRate ()
{
	if (audioMaster)
		audioMaster (&cEffect, audioMasterGetSampleRate, 0, 0, 0, 0);	// calls setSampleRate if implemented
	return sampleRate;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::updateBlockSize ()
{
	if (audioMaster)
		audioMaster (&cEffect, audioMasterGetBlockSize, 0, 0, 0, 0);	// calls setBlockSize if implemented
	return blockSize;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getInputLatency ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetInputLatency, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getOutputLatency ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetOutputLatency, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
AEffect* AudioEffectX::getPreviousPlug (long input)
{
	if (audioMaster)
		return (AEffect*) audioMaster (&cEffect, audioMasterGetPreviousPlug, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
AEffect* AudioEffectX::getNextPlug (long output)
{
	if (audioMaster)
		return (AEffect*) audioMaster (&cEffect, audioMasterGetNextPlug, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
// configuration
//-------------------------------------------------------------------------------------------------------

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::willProcessReplacing ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterWillReplaceOrAccumulate, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getCurrentProcessLevel ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetCurrentProcessLevel, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getAutomationState ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetAutomationState, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
void AudioEffectX::wantAsyncOperation (bool state)
{
	if (state)
		cEffect.flags |= effFlagsExtIsAsync;
	else
		cEffect.flags &= ~effFlagsExtIsAsync;
}

//-------------------------------------------------------------------------------------------------------
void AudioEffectX::hasExternalBuffer (bool state)
{
	if (state)
		cEffect.flags |= effFlagsExtHasBuffer;
	else
		cEffect.flags &= ~effFlagsExtHasBuffer;
}

//-------------------------------------------------------------------------------------------------------
// offline
//-------------------------------------------------------------------------------------------------------

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::offlineRead (VstOfflineTask* offline, VstOfflineOption option, bool readSource)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterOfflineRead, readSource, option, offline, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::offlineWrite (VstOfflineTask* offline, VstOfflineOption option)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterOfflineWrite, 0, option, offline, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::offlineStart (VstAudioFile* audioFiles, long numAudioFiles, long numNewAudioFiles)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterOfflineStart, numNewAudioFiles, numAudioFiles, audioFiles, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::offlineGetCurrentPass ()
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterOfflineGetCurrentPass, 0, 0, 0, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::offlineGetCurrentMetaPass ()
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterOfflineGetCurrentMetaPass, 0, 0, 0, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
// other
//-------------------------------------------------------------------------------------------------------


//-------------------------------------------------------------------------------------------------------
void AudioEffectX::setOutputSamplerate (float sampleRate)
{
	if (audioMaster)
		audioMaster (&cEffect, audioMasterSetOutputSampleRate, 0, 0, 0, sampleRate);
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::getSpeakerArrangement (VstSpeakerArrangement* pluginInput, VstSpeakerArrangement* pluginOutput)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterGetSpeakerArrangement, 0, (long)pluginInput, pluginOutput, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::getHostVendorString (char* text)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterGetVendorString, 0, 0, text, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::getHostProductString (char* text)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterGetProductString, 0, 0, text, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getHostVendorVersion ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetVendorVersion, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::hostVendorSpecific (long lArg1, long lArg2, void* ptrArg, float floatArg)
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterVendorSpecific, lArg1, lArg2, ptrArg, floatArg);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::canHostDo (char* text)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterCanDo, 0, 0, text, 0) != 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
void AudioEffectX::isSynth (bool state)
{
	if (state)
		cEffect.flags |= effFlagsIsSynth;
	else
		cEffect.flags &= ~effFlagsIsSynth;
}

//-------------------------------------------------------------------------------------------------------
void AudioEffectX::noTail (bool state)
{
	if (state)
		cEffect.flags |= effFlagsNoSoundInStop;
	else
		cEffect.flags &= ~effFlagsNoSoundInStop;
}

//-------------------------------------------------------------------------------------------------------
long AudioEffectX::getHostLanguage ()
{
	if (audioMaster)
		return audioMaster (&cEffect, audioMasterGetLanguage, 0, 0, 0, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
void* AudioEffectX::openWindow (VstWindow* window)
{
	if (audioMaster)
		return (void*)audioMaster (&cEffect, audioMasterOpenWindow, 0, 0, window, 0);
	return 0;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::closeWindow (VstWindow* window)
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterCloseWindow, 0, 0, window, 0) != 0);
	return false;
}

//-------------------------------------------------------------------------------------------------------
void* AudioEffectX::getDirectory ()
{
	if (audioMaster)
		return (void*)(audioMaster (&cEffect, audioMasterGetDirectory, 0, 0, 0, 0));
	return 0;
}

//-------------------------------------------------------------------------------------------------------
bool AudioEffectX::updateDisplay()
{
	if (audioMaster)
		return (audioMaster (&cEffect, audioMasterUpdateDisplay, 0, 0, 0, 0)) ? true : false;
	return 0;
}
