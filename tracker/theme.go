package tracker

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
)

var fontCollection []text.FontFace = gofont.Collection()
var textShaper = text.NewCache(fontCollection)

var neutral = color.RGBA{R: 18, G: 18, B: 18, A: 255}
var light = color.RGBA{R: 128, G: 128, B: 128, A: 255}
var dark = color.RGBA{R: 15, G: 15, B: 15, A: 255}
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
var blue = color.RGBA{R: 127, G: 127, B: 255, A: 255}
var gray = color.RGBA{R: 133, G: 133, B: 133, A: 255}
var darkGray = color.RGBA{R: 18, G: 18, B: 18, A: 255}
var black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
var yellow = color.RGBA{R: 255, G: 255, B: 130, A: 255}
var red = color.RGBA{R: 255, G: 0, B: 0, A: 255}

var transparent = color.RGBA{A: 0}

var primaryColorLight = color.RGBA{R: 243, G: 229, B: 245, A: 255}
var primaryColor = color.RGBA{R: 206, G: 147, B: 216, A: 255}
var primaryColorDark = color.RGBA{R: 123, G: 31, B: 162, A: 255}

var secondaryColorLight = color.RGBA{R: 224, G: 247, B: 250, A: 255}
var secondaryColor = color.RGBA{R: 128, G: 222, B: 234, A: 255}
var secondaryColorDark = color.RGBA{R: 0, G: 151, B: 167, A: 255}

var disabledContainerColor = color.RGBA{R: 31, G: 31, B: 31, A: 31}
var focusedContainerColor = color.RGBA{R: 31, G: 31, B: 31, A: 31}

var highEmphasisTextColor = color.RGBA{R: 222, G: 222, B: 222, A: 222}
var mediumEmphasisTextColor = color.RGBA{R: 153, G: 153, B: 153, A: 153}
var disabledTextColor = color.RGBA{R: 97, G: 97, B: 97, A: 97}

var panelColor = neutral
var panelShadeColor = neutral
var panelLightColor = light

var backgroundColor = color.RGBA{R: 18, G: 18, B: 18, A: 255}

var labelDefaultColor = highEmphasisTextColor
var labelDefaultBgColor = transparent
var labelDefaultFont = fontCollection[6].Font
var labelDefaultFontSize = unit.Sp(18)

var separatorLineColor = color.RGBA{R: 97, G: 97, B: 97, A: 97}

var activeTrackColor = focusedContainerColor
var trackSurfaceColor = color.RGBA{R: 31, G: 31, B: 31, A: 31}

var patternSurfaceColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}

var rowMarkerSurfaceColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
var rowMarkerPatternTextColor = secondaryColor
var rowMarkerRowTextColor = mediumEmphasisTextColor

var trackMenuSurfaceColor = color.RGBA{R: 31, G: 31, B: 31, A: 31}

var trackerFont = fontCollection[6].Font
var trackerFontSize = unit.Px(16)
var trackerInactiveTextColor = highEmphasisTextColor
var trackerTextColor = white
var trackerActiveTextColor = yellow
var trackerPatternRowTextColor = color.RGBA{R: 198, G: 198, B: 198, A: 255}
var trackerPlayColor = color.RGBA{R: 55, G: 55, B: 61, A: 255}
var trackerPatMarker = primaryColor
var trackerCursorColor = color.RGBA{R: 38, G: 79, B: 120, A: 64}
var trackerSelectionColor = color.RGBA{R: 19, G: 40, B: 60, A: 128}
var trackerSurfaceColor = color.RGBA{R: 18, G: 18, B: 18, A: 18}

var patternPlayColor = color.RGBA{R: 55, G: 55, B: 61, A: 255}
var patternTextColor = primaryColor
var patternActiveTextColor = yellow
var patternFont = fontCollection[6].Font
var patternFontSize = unit.Px(12)
var patternCursorColor = color.RGBA{R: 38, G: 79, B: 120, A: 64}
var patternSelectionColor = color.RGBA{R: 19, G: 40, B: 60, A: 128}

var inactiveBtnColor = color.RGBA{R: 61, G: 55, B: 55, A: 255}

var instrumentSurfaceColor = color.RGBA{R: 31, G: 31, B: 31, A: 31}

var songSurfaceColor = color.RGBA{R: 31, G: 31, B: 31, A: 31}
