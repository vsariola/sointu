package tracker

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
)

var fontCollection []text.FontFace = gofont.Collection()
var textShaper = text.NewCache(fontCollection)

var neutral = color.RGBA{R: 73, G: 117, B: 130, A: 255}
var light = color.RGBA{R: 138, G: 219, B: 243, A: 255}
var dark = color.RGBA{R: 24, G: 40, B: 44, A: 255}
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
var black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
var yellow = color.RGBA{R: 255, G: 255, B: 130, A: 255}
var red = color.RGBA{R: 255, G: 0, B: 0, A: 255}

var panelColor = neutral
var panelShadeColor = dark
var panelLightColor = light

var labelFont = fontCollection[6].Font
var labelFontSize = unit.Px(18)

var activeTrackColor = color.RGBA{0, 0, 50, 255}
var inactiveTrackColor = black

var trackerFont = fontCollection[6].Font
var trackerFontSize = unit.Px(16)
var trackerTextColor = white
var trackerActiveTextColor = yellow
var trackerPlayColor = red

var patternBgColor = black
var patternPlayColor = red
var patternTextColor = white
var patternActiveTextColor = yellow
var patternFont = fontCollection[6].Font
var patternFontSize = unit.Px(12)
