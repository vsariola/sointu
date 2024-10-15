package gioui

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
)

var fontCollection []text.FontFace = gofont.Collection()

var white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
var black = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
var transparent = color.NRGBA{A: 0}

var primaryColor = color.NRGBA{R: 206, G: 147, B: 216, A: 255}
var secondaryColor = color.NRGBA{R: 128, G: 222, B: 234, A: 255}

var highEmphasisTextColor = color.NRGBA{R: 222, G: 222, B: 222, A: 222}
var mediumEmphasisTextColor = color.NRGBA{R: 153, G: 153, B: 153, A: 153}
var disabledTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 97}

var backgroundColor = color.NRGBA{R: 18, G: 18, B: 18, A: 255}

var labelDefaultFont = fontCollection[6].Font
var labelDefaultFontSize = unit.Sp(18)

var rowMarkerPatternTextColor = secondaryColor
var rowMarkerRowTextColor = mediumEmphasisTextColor

var trackerFont = fontCollection[6].Font
var trackerFontSize = unit.Sp(16)
var trackerInactiveTextColor = highEmphasisTextColor
var trackerActiveTextColor = color.NRGBA{R: 255, G: 255, B: 130, A: 255}
var trackerPlayColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var trackerPatMarker = primaryColor
var oneBeatHighlight = color.NRGBA{R: 31, G: 37, B: 38, A: 255}
var twoBeatHighlight = color.NRGBA{R: 31, G: 51, B: 53, A: 255}

var patternPlayColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var patternTextColor = primaryColor
var patternCellColor = color.NRGBA{R: 255, G: 255, B: 255, A: 3}
var loopMarkerColor = color.NRGBA{R: 252, G: 186, B: 3, A: 255}

var instrumentHoverColor = color.NRGBA{R: 30, G: 31, B: 38, A: 255}
var instrumentNameHintColor = color.NRGBA{R: 200, G: 200, B: 200, A: 255}

var songSurfaceColor = color.NRGBA{R: 37, G: 37, B: 38, A: 255}

var popupSurfaceColor = color.NRGBA{R: 50, G: 50, B: 51, A: 255}
var popupShadowColor = color.NRGBA{R: 0, G: 0, B: 0, A: 192}

var dragListSelectedColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var dragListHoverColor = color.NRGBA{R: 42, G: 45, B: 61, A: 255}

var inactiveLightSurfaceColor = color.NRGBA{R: 37, G: 37, B: 38, A: 255}
var activeLightSurfaceColor = color.NRGBA{R: 45, G: 45, B: 45, A: 255}

var cursorColor = color.NRGBA{R: 100, G: 140, B: 255, A: 48}
var selectionColor = color.NRGBA{R: 100, G: 140, B: 255, A: 12}
var inactiveSelectionColor = color.NRGBA{R: 140, G: 140, B: 140, A: 16}

var errorColor = color.NRGBA{R: 207, G: 102, B: 121, A: 255}

var menuHoverColor = color.NRGBA{R: 30, G: 31, B: 38, A: 255}

var scrollBarColor = color.NRGBA{R: 255, G: 255, B: 255, A: 32}

var warningColor = color.NRGBA{R: 251, G: 192, B: 45, A: 255}

var dialogBgColor = color.NRGBA{R: 0, G: 0, B: 0, A: 224}
