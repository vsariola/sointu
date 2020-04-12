//-----------------------------------------------------------------------------
// VST Plug-Ins SDK
// User interface framework for VST plugins
//
// Version 1.0
//
// First version       : Wolfgang Kundrus
// Added Motif/Windows version : Yvan Grabit      01.98
// Added Mac version   : Charlie Steinberg        02.98
// Added BeOS version  : Georges-Edouard Berenger 05.99
//
// (c)1999 Steinberg Soft+Hardware GmbH
//-----------------------------------------------------------------------------

#ifndef __vstgui__
#define __vstgui__

// define global defines
#if WIN32
	#define WINDOWS 1
#elif SGI | SUN
	#define MOTIF 1
#elif __MWERKS__
	#define MAC 1
#endif

#ifndef __AEffEditor__
#include "AEffEditor.hpp"
#endif

//----------------------------------------------------
#if WINDOWS
	#include <windows.h>

//----------------------------------------------------
#elif MOTIF
	#include <X11/Xlib.h>
	#include <X11/Intrinsic.h>
	#ifdef NOBOOL
		#ifndef bool
			typedef short bool;
		#endif
		#ifndef false
			static const bool false = 0; 
		#endif
		#ifndef true
			static const bool true = 1;
		#endif
	#endif

// definition of struct for XPixmap resources
	struct CResTableEntry {
		int id;
  		char **xpm;
	};

	typedef CResTableEntry CResTable[];
	extern CResTable xpmResources;

//----------------------------------------------------
#elif MAC
	#include <Quickdraw.h>
	class CCashedPict;

//----------------------------------------------------
#elif BEOS
	#include <View.h>
	class PlugView;
	class BBitmap;
	class BResources;

#endif

//----------------------------------------------------
//----------------------------------------------------
class CFrame;
class CDrawContext;
class COffscreenContext;
class CControl;
class CBitmap;


//-----------------------------------------------------------------------------
// AEffGUIEditor Declaration
//-----------------------------------------------------------------------------
class AEffGUIEditor : public AEffEditor
{
public :

	AEffGUIEditor (AudioEffect *effect);

	virtual ~AEffGUIEditor ();

	virtual void setParameter (long index, float value) { postUpdate (); } 
	virtual long getRect (ERect **rect);
	virtual long open (void *ptr);
	virtual void idle ();
	virtual void draw (ERect *rect);
#if MAC
	virtual long mouse (long x, long y);
#endif

	// get the current time (in ms)
	long getTicks ();

	// feedback to appli.
	void doIdleStuff ();

	// get the effect attached to this editor
	AudioEffect *getEffect () { return effect; }

//---------------------------------------
protected:
	ERect   rect;
	CFrame *frame;

	unsigned long lLastTicks;
																
private:
	short sInControlLoop;
};


//-----------------------------------------------------------------------------
// Structure CRect
//-----------------------------------------------------------------------------
struct CRect
{
	CRect (long left = 0, long top = 0, long right = 0, long bottom = 0)
	:	left (left), top (top), right (right), bottom (bottom) {}
	CRect (const CRect& r)
	:	left (r.left), top (r.top), right (r.right), bottom (r.bottom) {}
	CRect& operator () (long left, long top, long right, long bottom)
	{
		if (left < right)
			this->left = left, this->right = right;
		else
			this->left = right, this->right = left;
		if (top < bottom)
			this->top = top, this->bottom = bottom;
		else
			this->top = bottom, this->bottom = top;
		return *this;
	}

	long left;
	long top;
	long right;
	long bottom;
	inline long width ()  { return right - left; }
	inline long height () { return bottom - top; }

	void offset (long x, long y) { left += x; right += x;
                                 top += y; bottom += y; }
};

//-----------------------------------------------------------------------------
// Structure CPoint
//-----------------------------------------------------------------------------
struct CPoint
{
	CPoint (long h = 0, long v = 0) : h (h), v (v) {}
	CPoint& operator () (long h, long v) 
	{ this->h = h; this->v = v;
		return *this; }

	bool isInside (CRect& r)
	{ return h >= r.left && h <= r.right && v >= r.top && v <= r.bottom; } 

	long h;
	long v;
};

//-----------------------------------------------------------------------------
// Structure CColor
//-----------------------------------------------------------------------------
struct CColor
{
	CColor& operator () (unsigned char red,
						unsigned char green,
						unsigned char blue,
						unsigned char unused)
	{
		this->red   = red;
		this->green = green;
		this->blue  = blue;
		this->unused = unused;
		return *this; 
	}

	CColor& operator = (CColor newColor)
	{
		red   = newColor.red;
		green = newColor.green;
		blue  = newColor.blue;
		unused = newColor.unused;
		return *this; 
	}

	bool operator != (CColor newColor)
	{
		return (red != newColor.red ||
						green != newColor.green ||
						blue  != newColor.blue);
	}

	bool operator == (CColor newColor)
	{
		return (red == newColor.red &&
						green == newColor.green &&
						blue  == newColor.blue);
	}
	
	unsigned char red;
	unsigned char green;
	unsigned char blue;
	unsigned char unused;
};

// define some basic colors
static CColor kTransparentCColor = {255, 255, 255, 0};
static CColor kBlackCColor  = {0,     0,   0, 0};
static CColor kWhiteCColor  = {255, 255, 255, 0};
static CColor kGreyCColor   = {127, 127, 127, 0};
static CColor kRedCColor    = {255,   0,   0, 0};
static CColor kGreenCColor  = {0  , 255,   0, 0};
static CColor kBlueCColor   = {0  ,   0, 255, 0};
static CColor kYellowCColor = {255, 255,   0, 0};
static CColor kCyanCColor   = {255,   0, 255, 0};
static CColor kMagentaCColor= {0  , 255, 255, 0};

//-----------------------------------------------------------------------------
//-----------
// Font type
//-----------
enum CFont
{
	kSystemFont = 0,
	kNormalFontVeryBig,
	kNormalFontBig,
	kNormalFont,
	kNormalFontSmall,
	kNormalFontVerySmall,
	kSymbolFont,

	kNumStandardFonts
};

//-----------
// Line style
//-----------
enum CLineStyle
{
	kLineSolid = 0,
	kLineOnOffDash
};

//----------------------------
// Text alignment (Horizontal)
//----------------------------
enum CHoriTxtAlign
{
	kLeftText = 0,
	kCenterText,
	kRightText
};

//----------------------------
// Buttons Type (+modifiers)
//----------------------------
enum CButton
{
	kLButton =  1,
	kMButton =  2,
	kRButton =  4,
	kShift   =  8,
	kControl = 16,
	kAlt     = 32,
	kApple   = 64
};

//-----------------------------------------------------------------------------
// CDrawContext Declaration
//-----------------------------------------------------------------------------
class CDrawContext
{
public:
	CDrawContext (CFrame *frame, void *systemContext, void *window = 0);
	virtual ~CDrawContext ();	

	void moveTo (CPoint &point);
	void lineTo (CPoint &point);

	void polyLine (CPoint *point, int numberOfPoints);
	void fillPolygon (CPoint *point, int numberOfPoints);

	void drawRect (CRect &rect);
	void fillRect (CRect &rect);

	void drawArc (CRect &rect, CPoint &point1, CPoint &point2);
	void fillArc (CRect &rect, CPoint &point1, CPoint &point2);

	void drawEllipse (CRect &rect);
	void fillEllipse (CRect &rect);
	
	void drawPoint (CPoint &point, CColor color);
	
	void       setLineStyle (CLineStyle style);
	CLineStyle getLineStyle () { return lineStyle; }

	void   setLineWidth (int width);
	int    getLineWidth () { return frameWidth; }

	void   setFillColor  (CColor color);
	CColor getFillColor () { return fillColor; }

	void   setFrameColor (CColor color);
	CColor getFrameColor () { return frameColor; }

	void   setFontColor (CColor color);
	CColor getFontColor () { return fontColor; }
	void   setFont (CFont fontID, const int size = 0);

	void drawString (const char *string, CRect &rect, const short opaque = false,
					 const CHoriTxtAlign hAlign = kCenterText);

	int  getMouseButtons ();
	void getMouseLocation (CPoint &point);

#if MOTIF
	int getIndexColor (CColor color);
	Colormap getColormap ();
	Visual   *getVisual ();
	unsigned int getDepth ();

	static int nbNewColor;
#endif

	void *getWindow () { return window; }
	void setWindow (void *ptr)  { window = ptr; }
	void getLoc (CPoint &where) { where = penLoc; }

	//-------------------------------------------
protected:

	friend class CBitmap;
	friend class COffscreenContext;

	void   *getSystemContext () { return systemContext; }

	void   *systemContext;
	void   *window;
	CFrame *frame;

	int    fontSize;
	CColor fontColor;
	CPoint penLoc;

	int    frameWidth;
	CColor frameColor;
	CColor fillColor;
	CLineStyle lineStyle;

#if WINDOWS
	void *brush;
	void *pen;
	void *font;
	void *oldbrush;
	void *oldpen;
	void *oldfont;
	int  iPenStyle;

#elif MAC
	FontInfo fontInfoStruct;
	Pattern  fillPattern;

#elif MOTIF
	Display *display;

	XFontStruct *fontInfoStruct;
	CFont fontInfoId;

#elif BEOS
	BView*	plugView;
	BFont	font;
	void lineFromTo (CPoint& cstart, CPoint& cend);

#endif
};

//-----------------------------------------------------------------------------
// COffscreenContext Declaration
//-----------------------------------------------------------------------------
class COffscreenContext : public CDrawContext
{
public:
	COffscreenContext (CDrawContext *context, CBitmap *bitmap);
	COffscreenContext (CFrame *frame, long width, long height, const CColor backgroundColor = kBlackCColor);

	virtual ~COffscreenContext ();

	void transfert (CDrawContext *context, CRect destRect, CPoint srcOffset = CPoint (0, 0));
	inline int getWidth ()  { return width; }
	inline int getHeight () { return height; }

	//-------------------------------------------
protected:
	bool    destroyPixmap;
	CBitmap *bitmap;

	long    height;
	long    width;

	CColor  backgroundColor;

#if MOTIF
	Display *xdisplay;

#elif BEOS
	BBitmap *offscreenBitmap;
#endif
};


//-----------------------------------------------------------------------------
// CBitmap Declaration
//-----------------------------------------------------------------------------
class CBitmap
{
public:
	CBitmap (int resourceID);
	CBitmap (CFrame &frame, int width, int height);
	~CBitmap ();

	void draw (CDrawContext*, CRect &rect, const CPoint &offset = CPoint (0, 0));
	void drawTransparent (CDrawContext *context, CRect &rect, const CPoint &offset = CPoint (0, 0));

	inline int getWidth ()  { return width; }
	inline int getHeight () { return height; }

	void forget ();
	void remember ();

	void *getHandle () { return handle; }
	
#if BEOS
	static void closeResource ();
#endif

	//-------------------------------------------
protected:
	int  resourceID;
	void *handle;
	void *mask;
	int  width;
	int  height;

	int  nbReference;

#if MOTIF
	void *createPixmapFromXpm (CDrawContext *context);

	char    **dataXpm;
	Display *xdisplay;

#elif MAC
	CCashedPict	*pPict;

#elif BEOS
	static BResources *resourceFile;
	BBitmap    *bbitmap;
	bool				transparencySet;
#endif
};

//-----------------------------------------------------------------------------
// CView Declaration
//-----------------------------------------------------------------------------
class CView
{
public:
	CView (CRect &size);
	virtual ~CView ();

	void redraw ();
	virtual void draw (CDrawContext *context);
	virtual void mouse (CDrawContext *context, CPoint &where);
	virtual void update (CDrawContext *context);

	virtual void looseFocus ();
	virtual void takeFocus ();

	virtual void setTempOffscreen (COffscreenContext *tempOffscr);

	int getHeight () { return size.height (); }
	int getWidth ()  { return size.width (); }

	CFrame *getParent () { return parent; }

	//-------------------------------------------
protected:
	friend class CControl;
	friend class CFrame;
	CRect  size;
	CFrame *parent;
	COffscreenContext *tempOffscreen;
};

//-----------------------------------------------------------------------------
// CFrame Declaration
//-----------------------------------------------------------------------------
class CFrame : public CView
{
public:
	CFrame (CRect &size, void *systemWindow, AEffGUIEditor *editor);
	CFrame (CRect &size, char *title, AEffGUIEditor *editor, const int style = 0);
	
	~CFrame ();

	bool open (CPoint *point = 0);
	bool close ();
	bool isOpen () { return openFlag; }

	void draw (CDrawContext *context);
	void draw (CView *view = 0);
	void mouse (CDrawContext *context, CPoint &where);

	void update (CDrawContext *context);
	void idle ();
	void doIdleStuff () { if (editor) editor->doIdleStuff (); }

	bool getPosition (int &x, int &y);
	bool setSize (int width, int height);
	bool getSize (CRect *size);

	void     setBackground (CBitmap *background);
	CBitmap *getBackground () { return background; }

	virtual bool addView (CView *view);
	virtual bool removeView (CView *view);

	int   setModalView (CView *view);
 
#if WINDOWS
	void *getSystemWindow () { return hwnd;}
#elif BEOS
	void *getSystemWindow () { return plugView;}
#else
	void *getSystemWindow () { return systemWindow;}
#endif

	AEffGUIEditor *getEditor () { return editor; }

	void   setEditView (CView *view) { editView = view; }
	CView *getEditView () { return editView; }

#if MOTIF
	Colormap getColormap ()   { return colormap; }
	Visual  *getVisual ()     { return visual; }
	unsigned int getDepth ()  { return depth; }
	Display *getDisplay ()    { return display; }
	Window   getWindow ()     { return window; }
	void     freeGc ();

	Region   region;

	GC       gc;
	GC       getGC ()         { return gc; }
#else
	void    *getGC ()         { return 0; }
#endif

	//-------------------------------------------
protected:
	bool   initFrame (void *systemWin);

	AEffGUIEditor *editor;
	
	void    *systemWindow;
	CBitmap *background;
	int      viewCount;
	int      maxViews;
	CView   **views;
	CView   *modalView;
	CView   *editView;

	bool    firstDraw;
	bool    openFlag;

#if WINDOWS
	void    *hwnd;
	friend LONG WINAPI WindowProc (HWND hwnd, UINT message, WPARAM wParam, LPARAM lParam);

#elif MOTIF
	Colormap  colormap;
	Display  *display;
	Visual   *visual;
	Window    window;
	unsigned int depth;

	friend void _destroyCallback (Widget, XtPointer, XtPointer);

#elif BEOS
	PlugView *plugView;
#endif

	//-------------------------------------------
private:
	bool      addedWindow;
	void     *vstWindow;
};


// include the control object
#ifndef __vstcontrols__
#include "vstcontrols.h"
#endif


//-End--------------------------------------
#endif
