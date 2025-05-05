package gioui

import (
	_ "embed"
	"fmt"
	"image/color"

	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"gopkg.in/yaml.v2"
)

type Theme struct {
	Material       material.Theme
	FilledButton   ButtonStyle
	TextButton     ButtonStyle
	DisabledButton ButtonStyle
	MenuButton     ButtonStyle
	Oscilloscope   OscilloscopeStyle
	NumericUpDown  NumericUpDownStyle
	DialogTitle    LabelStyle
	DialogText     LabelStyle
	SongPanel      struct {
		RowHeader  LabelStyle
		RowValue   LabelStyle
		Expander   LabelStyle
		Version    LabelStyle
		ErrorColor color.NRGBA
		Bg         color.NRGBA
	}
	Alert struct {
		Warning PopupAlertStyle
		Error   PopupAlertStyle
		Info    PopupAlertStyle
	}
	NoteEditor struct {
		TrackTitle LabelStyle
		OrderRow   LabelStyle
		PatternRow LabelStyle
		Note       LabelStyle
		PatternNo  LabelStyle
		Unique     LabelStyle
		Loop       color.NRGBA
		Header     LabelStyle
	}
	Dialog struct {
		Bg    color.NRGBA
		Title LabelStyle
		Text  LabelStyle
	}
	OrderEditor struct {
		TrackTitle LabelStyle
		RowTitle   LabelStyle
		Cell       LabelStyle
		Loop       color.NRGBA
	}
	Menu struct {
		Text     LabelStyle
		ShortCut LabelStyle
	}
	InstrumentEditor struct {
		Octave            LabelStyle
		Voices            LabelStyle
		InstrumentComment LabelStyle
		UnitComment       LabelStyle
		InstrumentList    struct {
			Number    LabelStyle
			Name      LabelStyle
			NameMuted LabelStyle
		}
		UnitList struct {
			Name         LabelStyle
			NameDisabled LabelStyle
			Comment      LabelStyle
			Stack        LabelStyle
			Disabled     LabelStyle
			Warning      color.NRGBA
			Error        color.NRGBA
		}
	}
	UnitEditor struct {
		Hint          LabelStyle
		Chooser       LabelStyle
		ParameterName LabelStyle
	}
	Cursor struct {
		Active   color.NRGBA
		Inactive color.NRGBA
	}
	Selection struct {
		Active   color.NRGBA
		Inactive color.NRGBA
	}
}

//go:embed theme.yml
var defaultTheme []byte

func NewTheme() *Theme {
	var theme Theme
	err := yaml.Unmarshal(defaultTheme, &theme)
	if err != nil {
		panic(fmt.Errorf("failed to default theme: %w", err))
	}
	str, _ := yaml.Marshal(theme)
	fmt.Printf(string(str))
	ReadCustomConfigYml("theme.yml", &theme)
	theme.Material.Shaper = &text.Shaper{}
	theme.Material.Icon.CheckBoxChecked = mustIcon(widget.NewIcon(icons.ToggleCheckBox))
	theme.Material.Icon.CheckBoxUnchecked = mustIcon(widget.NewIcon(icons.ToggleCheckBoxOutlineBlank))
	theme.Material.Icon.RadioChecked = mustIcon(widget.NewIcon(icons.ToggleRadioButtonChecked))
	theme.Material.Icon.RadioUnchecked = mustIcon(widget.NewIcon(icons.ToggleRadioButtonUnchecked))
	return &theme
}

func mustIcon(ic *widget.Icon, err error) *widget.Icon {
	if err != nil {
		panic(err)
	}
	return ic
}

var white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
var black = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
var transparent = color.NRGBA{A: 0}

var mediumEmphasisTextColor = color.NRGBA{R: 153, G: 153, B: 153, A: 153}
var disabledTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 97}

var trackerPlayColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var oneBeatHighlight = color.NRGBA{R: 31, G: 37, B: 38, A: 255}
var twoBeatHighlight = color.NRGBA{R: 31, G: 51, B: 53, A: 255}

var patternPlayColor = color.NRGBA{R: 55, G: 55, B: 61, A: 255}
var patternCellColor = color.NRGBA{R: 255, G: 255, B: 255, A: 3}

var popupSurfaceColor = color.NRGBA{R: 50, G: 50, B: 51, A: 255}
var popupShadowColor = color.NRGBA{R: 0, G: 0, B: 0, A: 192}

var cursorForTrackMidiInColor = color.NRGBA{R: 255, G: 100, B: 140, A: 48}
var cursorNeighborForTrackMidiInColor = color.NRGBA{R: 255, G: 100, B: 140, A: 24}

var menuHoverColor = color.NRGBA{R: 30, G: 31, B: 38, A: 255}

var scrollBarColor = color.NRGBA{R: 255, G: 255, B: 255, A: 32}

var paramIsSendTargetColor = color.NRGBA{R: 120, G: 120, B: 210, A: 255}
var paramValueInvalidColor = color.NRGBA{R: 120, G: 120, B: 120, A: 190}
