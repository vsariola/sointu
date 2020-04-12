#ifndef __Go4kVSTi__
#define __Go4kVSTi__

#include <string.h>
#include <vector>

#ifndef __AudioEffectX__
#include "audioeffectx.h"
#endif

//------------------------------------------------------------------------------------------
class Go4kVSTi : public AudioEffectX
{
public:
	Go4kVSTi(audioMasterCallback audioMaster);
	~Go4kVSTi();

	virtual void process(float **inputs, float **outputs, long sampleframes);
	virtual void processReplacing(float **inputs, float **outputs, long sampleframes);
	void processAnyhow(float **inputs, float **outputs, long sampleFrames);
	virtual long processEvents(VstEvents* events);

	virtual void setProgram(long program);
	virtual void setProgramName(char *name);
	virtual void getProgramName(char *name);
	virtual void setParameter(long index, float value);
	virtual float getParameter(long index);
	virtual void getParameterLabel(long index, char *label);
	virtual void getParameterDisplay(long index, char *text);
	virtual void getParameterName(long index, char *text);
	virtual void setSampleRate(float sampleRate);
	virtual void setBlockSize(long blockSize);
	virtual void suspend();
	virtual void resume();
	virtual bool getOutputProperties (long index, VstPinProperties* properties);
	virtual bool getProgramNameIndexed (long category, long index, char* text);
	virtual bool copyProgram (long destination);
	virtual bool getEffectName (char* name);
	virtual bool getVendorString (char* text);
	virtual bool getProductString (char* text);
	virtual long getVendorVersion () {return 1;}
	virtual long canDo (char* text);
	virtual long getChunk(void** data, bool isPreset = false) override;
	virtual long setChunk(void* data, long byteSize, bool isPreset = false) override;

private:
	void initProcess();
	void ApplyEvent(VstMidiEvent *event);

	std::vector<VstMidiEvent*> m_currentEvents;

	unsigned char *m_chunkBuffer;
};

#endif
