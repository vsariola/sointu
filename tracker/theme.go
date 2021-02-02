package tracker

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
)

var fontCollection []text.FontFace = gofont.Collection()
var textShaper = text.NewCache(fontCollection)

var neutral = color.NRGBA{R: 18, G: 18, B: 18, A: 255}
var light = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
var dark = color.NRGBA{R: 15, G: 15, B: 15, A: 255}
var white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
var blue = color.NRGBA{R: 127, G: 127, B: 255, A: 255}
var gray = color.NRGBA{R: 133, G: 133, B: 133, A: 255}
var darkGray = color.NRGBA{R: 18, G: 18, B: 18, A: 255}
var black = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
var yellow = color.NRGBA{R: 255, G: 255, B: 130, A: 255}
var red = color.NRGBA{R: 255, G: 0, B: 0, A: 255}

var transparent = color.NRGBA{A: 0}

var primaryColorLight = color.NRGBA{R: 243, G: 229, B: 245, A: 255}
var primaryColor = color.NRGBA{R: 206, G: 147, B: 216, A: 255}
var primaryColorDark = color.NRGBA{R: 123, G: 31, B: 162, A: 255}

var secondaryColorLight = color.NRGBA{R: 224, G: 247, B: 250, A: 255}
var secondaryColor = color.NRGBA{R: 128, G: 222, B: 234, A: 255}
var secondaryColorDark = color.NRGBA{R: 0, G: 151, B: 167, A: 255}

var disabledContainerColor = color.NRGBA{R: 255, G: 255, B: 255, A: 5}
var focusedContainerColor = color.NRGBA{R: 255, G: 255, B: 255, A: 5}

var highEmphasisTextColor = color.NRGBA{R: 222, G: 222, B: 222, A: 222}
var mediumEmphasisTextColor = color.NRGBA{R: 153, G: 153, B: 153, A: 153}
var disabledTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 97}

var panelColor = neutral
var panelShadeColor = neutral
var panelLightColor = light

var backgroundColor = color.NRGBA{R: 18, G: 18, B: 18, A: 255}

var labelDefaultColor = highEmphasisTextColor
var labelDefaultBgColor = transparent
var labelDefaultFont = fontCollection[6].Font
var labelDefaultFontSize = unit.Sp(18)

var separatorLineColor = color.NRGBA{R: 255, G: 255, B: 255, A: 97}

var activeTrackColor = focusedContainerColor
var trackSurfaceColor = color.NRGBA{R: 255, G: 255, B: 255, A: 31}

var patternSurfaceColor = color.NRGBA{R: 24, G: 24, B: 24, A: 255}

var rowMarkerSurfaceColor = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
var rowMarkerPatternTextColor = secondaryColor
var rowMarkerRowTextColor = mediumEmphasisTextColor

var trackMenuSurfaceColor = color.NRGBA{R: 37, G: 37, B: 38, A: 255}

var trackerFont = fontCollection[6].Font
var trackerFontSize = unit.Px(16)
var trackerInactiveTextColor = highEmphasisTextColor
var trackerTextColor = white
var trackerActiveTextColor = yellow
var trackerPatternRowTextColor = color.NRGBA{R: 198, G: 198, B: 198, A: 255}
var trackerPlayColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var trackerPatMarker = primaryColor
var trackerCursorColor = color.NRGBA{R: 100, G: 140, B: 255, A: 48}
var trackerSelectionColor = color.NRGBA{R: 100, G: 140, B: 255, A: 8}
var trackerSurfaceColor = color.NRGBA{R: 30, G: 30, B: 30, A: 255}

var patternPlayColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var patternTextColor = primaryColor
var patternActiveTextColor = yellow
var patternFont = fontCollection[6].Font
var patternFontSize = unit.Px(12)
var patternCursorColor = color.NRGBA{R: 38, G: 79, B: 120, A: 64}
var patternSelectionColor = color.NRGBA{R: 19, G: 40, B: 60, A: 128}

var inactiveBtnColor = color.NRGBA{R: 61, G: 55, B: 55, A: 255}

var instrumentSurfaceColor = color.NRGBA{R: 37, G: 37, B: 38, A: 255}

var songSurfaceColor = color.NRGBA{R: 37, G: 37, B: 38, A: 255}
