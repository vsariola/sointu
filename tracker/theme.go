package tracker

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
)

var fontCollection []text.FontFace = gofont.Collection()
var textShaper = text.NewCache(fontCollection)

var neutral = color.RGBA{R: 64, G: 64, B: 64, A: 255}
var light = color.RGBA{R: 128, G: 128, B: 128, A: 255}
var dark = color.RGBA{R: 15, G: 15, B: 15, A: 255}
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
var blue = color.RGBA{R: 127, G: 127, B: 255, A: 255}
var gray = color.RGBA{R: 133, G: 133, B: 133, A: 255}
var darkGray = color.RGBA{R: 30, G: 30, B: 30, A: 255}
var black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
var yellow = color.RGBA{R: 255, G: 255, B: 130, A: 255}
var red = color.RGBA{R: 255, G: 0, B: 0, A: 255}

var panelColor = neutral
var panelShadeColor = neutral
var panelLightColor = light

var labelFont = fontCollection[6].Font
var labelFontSize = unit.Px(18)

var activeTrackColor = color.RGBA{R: 45, G: 45, B: 45, A: 255}
var inactiveTrackColor = darkGray

var trackerFont = fontCollection[6].Font
var trackerFontSize = unit.Px(16)
var trackerInactiveTextColor = color.RGBA{R: 212, G: 212, B: 212, A: 255}
var trackerTextColor = white
var trackerActiveTextColor = yellow
var trackerPatternRowTextColor = color.RGBA{R: 198, G: 198, B: 198, A: 255}
var trackerPlayColor = color.RGBA{R: 55, G: 55, B: 61, A: 255}
var trackerPatMarker = blue
var trackerCursorColor = color.RGBA{R: 38, G: 79, B: 120, A: 64}

var patternBgColor = black
var patternPlayColor = color.RGBA{R: 55, G: 55, B: 61, A: 255}
var patternTextColor = white
var patternActiveTextColor = yellow
var patternFont = fontCollection[6].Font
var patternFontSize = unit.Px(12)
