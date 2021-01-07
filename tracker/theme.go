package tracker

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
)

var fontCollection []text.FontFace = gofont.Collection()
var textShaper = text.NewCache(fontCollection)

var neutral = color.RGBA{R: 64, G: 39, B: 132, A: 255}
var light = color.RGBA{R: 117, G: 75, B: 234, A: 255}
var dark = color.RGBA{R: 25, G: 15, B: 51, A: 255}
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
var blue = color.RGBA{R: 127, G: 127, B: 255, A: 255}
var gray = color.RGBA{R: 127, G: 127, B: 127, A: 255}
var black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
var yellow = color.RGBA{R: 255, G: 255, B: 130, A: 255}
var red = color.RGBA{R: 255, G: 0, B: 0, A: 255}

var panelColor = neutral
var panelShadeColor = neutral
var panelLightColor = light

var labelFont = fontCollection[6].Font
var labelFontSize = unit.Px(18)

var activeTrackColor = color.RGBA{0, 0, 50, 255}
var inactiveTrackColor = black

var trackerFont = fontCollection[6].Font
var trackerFontSize = unit.Px(16)
var trackerInactiveTextColor = gray
var trackerTextColor = white
var trackerActiveTextColor = yellow
var trackerPlayColor = red
var trackerPatMarker = blue
var trackerCursorColor = color.RGBA{R: 64, G: 64, B: 64, A: 64}

var patternBgColor = black
var patternPlayColor = red
var patternTextColor = white
var patternActiveTextColor = yellow
var patternFont = fontCollection[6].Font
var patternFontSize = unit.Px(12)
