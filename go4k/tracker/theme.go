package tracker

import (
	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
	"image/color"
)

var fontCollection []text.FontFace = gofont.Collection()
var textShaper = text.NewCache(fontCollection)

var neutral = color.RGBA{R: 73, G: 117, B: 130, A: 255}
var light = color.RGBA{R: 138, G: 219, B: 243, A: 255}
var dark = color.RGBA{R: 24, G: 40, B: 44, A: 255}
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
var black = color.RGBA{R: 0, G: 0, B: 0, A: 255}

var panelColor = neutral
var panelShadeColor = dark
var panelLightColor = light

var labelFont = fontCollection[6].Font
var labelFontSize = unit.Px(18)
