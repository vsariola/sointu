//-----------------------------------------------------------------------------
// VST Plug-Ins SDK
// Simple user interface framework for VST plugins
// Standard control objects
//
// Version 1.0
//
// First version       : Wolfgang Kundrus
// Added new objects   : Michael Schmidt          08.97
// Added new objects   : Yvan Grabit              01.98
//
// (c)1999 Steinberg Soft+Hardware GmbH
//-----------------------------------------------------------------------------

#ifndef __vstcontrols__
#define __vstcontrols__

#ifndef __vstgui__
#include "vstgui.h"
#endif

//------------------
// defines
//------------------
#ifndef kPI
#define kPI 3.14159265358979323846
#endif

#ifndef k2PI
#define k2PI 6.28318530717958647692
#endif

#ifndef kPI_2
#define kPI_2 1.57079632679489661923f
#endif
#ifndef kPI_4
#define kPI_4 0.78539816339744830962
#endif

#ifndef kE
#define kE 2.7182818284590452354
#endif

#ifndef kLN2
#define kLN2 0.69314718055994530942
#endif


//------------------
// CControlEnum type
//------------------
enum CControlEnum
{
	kHorizontal = 1 << 0,
	kVertical   = 1 << 1,
	kShadowText = 1 << 2,
	kLeft       = 1 << 3,
	kRight      = 1 << 4,
	kTop        = 1 << 5,
	kBottom     = 1 << 6,
	k3DIn       = 1 << 7,
	k3DOut      = 1 << 8,
	kPopupStyle = 1 << 9,
	kCheckStyle = 1 << 10
};

//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CControlListener
{
public:
	virtual void valueChanged (CDrawContext *context, CControl *control) = 0;
};

//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CControl : public CView
{
public:
	CControl (CRect &size, CControlListener *listener, int tag);
	virtual ~CControl ();

	virtual void  draw (CDrawContext *context) = 0;
	virtual	void  update (CDrawContext *context);
	virtual void  doIdleStuff () { if (parent) parent->doIdleStuff (); }

	virtual void  setValue (float val) { value = val; }
	virtual float getValue () { return value; };

	virtual void  setMin (float val) { vmin = val; }
	virtual float getMin () { return vmin; }
	virtual void  setMax (float val) { vmax = val; }
	virtual float getMax () { return vmax; }

	virtual void  setOldValue (float val) { oldValue = val; }
	virtual	float getOldValue (void) { return oldValue; }
	virtual void  setDefaultValue (float val) { defaultValue = val; }
	virtual	float getDefaultValue (void) { return defaultValue; }

	inline  int   getTag () { return tag; }

	virtual void  setMouseEnabled (bool bEnable = true) { bMouseEnabled = bEnable; }
	virtual bool  getMouseEnabled () { return bMouseEnabled; }

protected:
	CControlListener *listener;
	long  tag;
	bool  dirty;
	bool  bMouseEnabled;
	float oldValue;
	float defaultValue;
	float value;
	float vmin;
	float vmax;
	float step;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class COnOffButton : public CControl
{
public:
	COnOffButton (CRect &size, CControlListener *listener, int tag,
                  CBitmap *handle);
	virtual ~COnOffButton ();

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

protected:
	CBitmap *handle;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CParamDisplay : public CControl
{
public:
	CParamDisplay (CRect &size, CBitmap *background = 0, int style = 0);
	virtual ~CParamDisplay ();
	
	virtual void setFont (CFont fontID);
	virtual void setFontColor (CColor color);
	virtual void setBackColor (CColor color);
	virtual void setFrameColor (CColor color);
	virtual void setShadowColor (CColor color);

	virtual void setHoriAlign (CHoriTxtAlign hAlign);
	virtual void setBackOffset (CPoint &offset);
	virtual void setStringConvert (void (*stringConvert) (float value, char *string));

	virtual void draw (CDrawContext *context);

protected:
	void drawText (CDrawContext *context, char *string, CBitmap *newBack = 0);

	CHoriTxtAlign horiTxtAlign;
	int     style;

	CFont   fontID;
	CColor  fontColor;
	CColor  backColor;
	CColor  frameColor;
	CColor  shadowColor;
	CPoint  offset;

	CBitmap *background;

private:
	void (*stringConvert) (float value, char *string);
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CTextEdit : public CParamDisplay
{
public:
	CTextEdit (CRect &size, CControlListener *listener, int tag, const char *txt = 0,
               CBitmap *background = 0,
               int style = 0);
	~CTextEdit ();

	virtual void setText (char *txt);
	virtual void getText (char *txt);

	virtual	void draw (CDrawContext *context);
	virtual	void mouse (CDrawContext *context, CPoint &where);

	virtual void setTextEditConvert (void (*stringConvert) (char *input, char *string));

	virtual	void takeFocus ();
	virtual	void looseFocus ();

protected:
	void *platformControl;
	void *platformFont;
	char text[256];

private:
	void (*stringConvert) (char *input, char *string);
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
#define MAX_ENTRY 128
 class COptionMenu : public CParamDisplay
{
public:
	COptionMenu (CRect &size, CControlListener *listener, int tag,
                 CBitmap *background = 0, CBitmap *bgWhenClick = 0,
                 int style = 0);
	~COptionMenu ();

	virtual	bool addEntry (char *txt, int index = -1);
	virtual	int  getCurrent (char *txt = 0);
	virtual	bool setCurrent (int index);
	virtual	bool getEntry (int index, char *txt);
	virtual	bool removeEntry (int index);
	virtual	bool removeAllEntry ();
	virtual int  getNbEntries () { return nbEntries; }

	virtual	void draw (CDrawContext *context);
	virtual	void mouse (CDrawContext *context, CPoint &where);

	virtual	void takeFocus ();
	virtual	void looseFocus ();

#if MOTIF
	void setCurrentSelected (void *itemSelected);
#endif

protected:
	void    *platformControl;
	char    *entry[MAX_ENTRY];

#if MOTIF
	void    *itemWidget[MAX_ENTRY];
#endif

	int      nbEntries;
	int      currentIndex;
	CBitmap *bgWhenClick;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CKnob : public CControl
{
public:
	CKnob (CRect &size, CControlListener *listener, int tag, 
           CBitmap *background,
           CBitmap *handle, CPoint &offset);
	virtual ~CKnob ();

	virtual void draw (CDrawContext*);
	virtual	void mouse (CDrawContext *context, CPoint &where);

	virtual void  valueToPoint (CPoint &point);
	virtual float valueFromPoint (CPoint &point);

	virtual void     setBackground (CBitmap* background);
	virtual CBitmap *getBackground () { return background; }

protected:
	int      inset;
	CBitmap *background;
	CBitmap *handle;
	CPoint   offset;
};

//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CAnimKnob : public CKnob
{
public:
	CAnimKnob (CRect &size, CControlListener *listener, int tag, 
               int subPixmaps,        // number of subPixmaps
               int heightOfOneImage,  // pixel
               CBitmap *handle, CPoint &offset);
	virtual ~CAnimKnob ();

	virtual void draw (CDrawContext*);

protected:
	int subPixmaps;		// number of subPixmaps
	int heightOfOneImage;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CVerticalSwitch : public CControl
{
public:
	CVerticalSwitch (CRect &size, CControlListener *listener, int tag, 
                     int subPixmaps,         // number of subPixmaps
                     int heightOfOneImage,   // pixel
                     int iMaxPositions,
                     CBitmap *handle, CPoint &offset);
	virtual ~CVerticalSwitch ();

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

protected:
	CBitmap *handle;
	CPoint   offset;
	int      subPixmaps;            // number of subPixmaps
	int      heightOfOneImage;
	int      iMaxPositions;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CHorizontalSwitch : public CControl
{
public:
	CHorizontalSwitch (CRect &size, CControlListener *listener, int tag, 
                       int subPixmaps,        // number of subPixmaps
                       int heightOfOneImage,  // pixel
                       int iMaxPositions,
                       CBitmap *handle,
                       CPoint &offset);
	virtual	~CHorizontalSwitch ();

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

protected:
	CBitmap *handle;
	CPoint   offset;
	int      subPixmaps;        // number of subPixmaps
	int      heightOfOneImage;
	int      iMaxPositions;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CRockerSwitch : public CControl
{
public:
	CRockerSwitch (CRect &size, CControlListener *listener, int tag, 
                   int heightOfOneImage,  // pixel
                   CBitmap *handle, CPoint &offset);
	virtual ~CRockerSwitch ();

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

protected:
	CBitmap *handle;
	CPoint   offset;
	int      heightOfOneImage;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CMovieBitmap : public CControl
{
public:
	CMovieBitmap (CRect &size, CControlListener *listener, int tag, 
                  int subPixmaps,        // number of subPixmaps
                  int heightOfOneImage,  // pixel
                  CBitmap *handle, CPoint &offset);
	virtual	~CMovieBitmap ();

	virtual void draw (CDrawContext*);

protected:
	CBitmap *handle;
	CPoint   offset;
	int      subPixmaps;         // number of subPixmaps
	int      heightOfOneImage;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CMovieButton : public CControl
{
public:
	CMovieButton (CRect &size, CControlListener *listener, int tag, 
                  int heightOfOneImage,  // pixel
                  CBitmap *handle, CPoint &offset);
	virtual ~CMovieButton ();	

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

protected:
	CBitmap *handle;
	CPoint   offset;
	int      heightOfOneImage;
	float    buttonState;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
// displays bitmaps within a (child-) window
class CAutoAnimation : public CControl
{
public:
	CAutoAnimation (CRect &size, CControlListener *listener, int tag, 
                    int subPixmaps,        // number of subPixmaps...
                    int heightOfOneImage,  // pixel
                    CBitmap *handle, CPoint &offset);
	virtual ~CAutoAnimation ();

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

	virtual void openWindow (void);
	virtual void closeWindow (void);

	virtual void nextPixmap (void);
	virtual void previousPixmap (void);

	bool    isWindowOpened () { return windowOpened; }

protected:
	CBitmap *handle;
	CPoint   offset;

	int      subPixmaps;
	int      heightOfOneImage;

	bool     windowOpened;
	int      totalHeightOfBitmap;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
// Vertical Slider
class CVerticalSlider : public CControl
{
public:
	CVerticalSlider (CRect &size, CControlListener *listener, int tag, 
                     int     iMinYPos,    // min Y position in pixel
                     int     iMaxYPos,    // max Y position in pixel
                     CBitmap *handle,     // bitmap slider
                     CBitmap *background, // bitmap background
                     CPoint  &offset,
                     int     style = kBottom); // style (kBottom, kTop))

	virtual ~CVerticalSlider ();
  
	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

	virtual void setDrawTransparentHandle (bool val) { drawTransparentEnabled = val; }
	virtual void setOffsetHandle (CPoint &val) { offsetHandle = val; }

protected:
	CBitmap *handle;
	CBitmap *background;

	int      widthOfSlider; // size of the handle-slider
	int      heightOfSlider;

	CPoint   offset; 
	CPoint   offsetHandle;

	int      iMinYPos;     // min Y position in pixel
	int      iMaxYPos;     // max Y position in pixel
	int      style;

	int      actualYPos;
	bool     drawTransparentEnabled;

	int      minTmp;
	int      maxTmp;
	int      widthControl;
	int      heightControl;
};

//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
// Horizontal Slider
class CHorizontalSlider : public CControl
{
public:
	CHorizontalSlider (CRect &size, CControlListener *listener, int tag, 
                       int     iMinXPos,    // min X position in pixel
                       int     iMaxXPos,    // max X position in pixel
                       CBitmap *handle,     // bitmap slider
                       CBitmap *background, // bitmap background	
                       CPoint  &offset,
                       int     style = kRight); // style (kRight, kLeft));
  
	virtual ~CHorizontalSlider ();
  
	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

	virtual void setDrawTransparentHandle (bool val) { drawTransparentEnabled = val; }
	virtual void setOffsetHandle (CPoint &val) { offsetHandle = val; }

protected:
	CBitmap *handle;
	CBitmap *background;

	int      widthOfSlider;      // size of the handle-slider
	int      heightOfSlider;

	CPoint   offset; 
 	CPoint   offsetHandle;
 
	int      iMinXPos;     // min X position in pixel
	int      iMaxXPos;     // max X position in pixel
	int      style;

	int      actualXPos;
	bool     drawTransparentEnabled;

	int      minTmp;
	int      maxTmp;
	int      widthControl;
	int      heightControl;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
// special display with custom numbers (0...9)
class CSpecialDigit : public CControl
{
public:
	CSpecialDigit (CRect &size, CControlListener *listener, int tag, // tag identifier
                   long     dwPos,     // actual value
                   int      iNumbers,  // amount of numbers (max 7)
                   int      *xpos,     // array of all XPOS
                   int      *ypos,     // array of all YPOS
                   int      width,     // width of ONE number
                   int      height,    // height of ONE number
                   CBitmap  *handle);  // bitmap numbers
	virtual ~CSpecialDigit ();
	
	virtual void  draw (CDrawContext*);

	virtual float getNormValue (void);

protected:
	CBitmap *handle;
	int      iNumbers;   // amount of numbers
	int      xpos[7];    // array of all XPOS, max 7 possible
	int      ypos[7];    // array of all YPOS, max 7 possible
	int      width;      // width  of ONE number
	int      height;     // height of ONE number
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CKickButton : public CControl
{
public:
	CKickButton (CRect &size, CControlListener *listener, int tag, 
                 int heightOfOneImage,  // pixel
                 CBitmap *handle, CPoint &offset);
	virtual ~CKickButton ();	

	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);

protected:
	CBitmap *handle;
	CPoint   offset;
	int      heightOfOneImage;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CSplashScreen : public CControl
{
public:
	CSplashScreen (CRect &size, CControlListener *listener, int tag, 
                   CBitmap *handle,
                   CRect &toDisplay, 
                   CPoint &offset);
	virtual ~CSplashScreen ();	
  
	virtual void draw (CDrawContext*);
	virtual void mouse (CDrawContext *context, CPoint &where);
  
protected:
	CRect    toDisplay;
	CRect    keepSize;
	CBitmap *handle;
	CPoint   offset;
};


//-----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
class CVuMeter : public CControl
{
public:
	CVuMeter (CRect& size, CBitmap *onBitmap, CBitmap *offBitmap,
              int nbLed, const int style = kVertical);
	virtual ~CVuMeter ();	
  
	virtual void setDecreaseStepValue (float value) { decreaseValue = value; }

	virtual void draw (CDrawContext *context);

protected:
	CBitmap *onBitmap;
	CBitmap *offBitmap;
	int      nbLed;
	int      style;
	float    decreaseValue;

	CRect    rectOn;
	CRect    rectOff;
};

#endif
