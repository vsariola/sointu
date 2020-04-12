#ifndef __Go4kVSTi__
#include "Go4kVSTi.h"
#include "Go4kVSTiCore.h"
#include "stdio.h"
#endif

#include "math.h"

//-----------------------------------------------------------------------------------------
void Go4kVSTi::setSampleRate (float sampleRate)
{
	AudioEffectX::setSampleRate (sampleRate);
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::setBlockSize (long blockSize)
{
	AudioEffectX::setBlockSize (blockSize);
	// you may need to have to do something here...
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::suspend ()
{
	m_currentEvents.clear();
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::resume ()
{
	wantEvents ();
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::initProcess ()
{
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::process (float **inputs, float **outputs, long sampleFrames)
{
	processAnyhow(inputs, outputs, sampleFrames);	
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::processReplacing (float **inputs, float **outputs, long sampleFrames)
{
	processAnyhow(inputs, outputs, sampleFrames);
}

void Go4kVSTi::processAnyhow(float **inputs, float **outputs, long sampleFrames)
{
	float* outl = outputs[0];
	float* outr = outputs[1];
	static int bpmCheck = 0;
	if ( bpmCheck <= 0 )
	{
		VstTimeInfo* myTime = getTimeInfo ( kVstTempoValid );
		Go4kVSTi_SetBPM(myTime->tempo);
		bpmCheck = 20; // only check every 20th call of this function (bpm shouldnt change that often)
	}
	bpmCheck--;

	int start = 0;
	// midi events will occur this frame, so render partially
	if (m_currentEvents.size() > 0)
	{		
		// for all events
		for (unsigned int i = 0; i < m_currentEvents.size(); i++)
		{
			// process samples until next event
			int todo = m_currentEvents[i]->deltaFrames - start;
			Go4kVSTi_Tick(outl+start, outr+start, todo);
			start = m_currentEvents[i]->deltaFrames;
			// apply changes due to event
			ApplyEvent(m_currentEvents[i]);
		}
	}
	Go4kVSTi_Tick(outl+start, outr+start, sampleFrames - start);

	// clear event list (old event pointer wont be valid next frame anyway)
	m_currentEvents.clear();
}

//-----------------------------------------------------------------------------------------
long Go4kVSTi::processEvents(VstEvents* ev)
{
	for (long i = 0; i < ev->numEvents; i++)
	{
		if ((ev->events[i])->type != kVstMidiType)
			continue;
		VstMidiEvent* event = (VstMidiEvent*)ev->events[i];
		m_currentEvents.push_back(event);
	} 
	return 1;	// want more
}

//-----------------------------------------------------------------------------------------
void Go4kVSTi::ApplyEvent(VstMidiEvent *event)
{
	char* midiData = event->midiData;
	byte status = midiData[0] & 0xf0;		// status
	byte channel = midiData[0] & 0x0f;		// channel

	// note on/off events
	if (status == 0x90 || status == 0x80)
	{
		byte note = midiData[1] & 0x7f;
		byte velocity = midiData[2] & 0x7f;
		if (status == 0x80)
			velocity = 0;
		// note off
		if (!velocity)
		{
			Go4kVSTi_StopVoice(channel, note);
		}
		// note on	
		else
		{
			Go4kVSTi_AddVoice(channel, note);
		}
	}
/*	// polyphonic aftertouch
	else if (status == 0xA)
	{
		byte note = midiData[1] & 0x7f;
		byte pressure = midiData[2] & 0x7f;
		Go4kVSTi_PolyAftertouch(channel, note, pressure);
	}
	// channel aftertouch
	else if (status == 0xD)
	{
		byte pressure = midiData[1] & 0x7f;
		Go4kVSTi_ChannelAftertouch(channel, pressure);
	}
	// Controller Change
	else if (status == 0xB0)
	{
		byte number = midiData[1] & 0x7f;
		byte value = midiData[2] & 0x7f;
//		Go4kVSTi_ControllerChange(channel, number, value, event->deltaFrames);
	}
	// Pitch Bend
	else if (status == 0xE0)
	{
		byte lsb = midiData[1] & 0x7f;
		byte msb = midiData[2] & 0x7f;
		int value = (((int)(msb)) << 7) + lsb;
//		Go4kVSTi_PitchBend(channel, value); // 0 - 16383, center 8192
		// dont use full precision for the sake of equally sized streams
//		Go4kVSTi_PitchBend(channel, value >> 7, event->deltaFrames); // 0 - 127, center 64
	}*/
	// all notes off (dont seem to come anyway
	else if (status == 0xb0 && midiData[1] == 0x7e)	// all notes off
	{
		Go4kVSTi_StopVoice(channel, 0);
	}		
}
