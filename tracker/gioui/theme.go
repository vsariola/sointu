package gioui

import (
	_ "embed"
	"image/color"

	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type Theme struct {
	Define   any // this is just needed for yaml.UnmarshalStrict, so we can have "defines" in the yaml
	Material material.Theme
	Button   struct {
		Filled   ButtonStyle
		Text     ButtonStyle
		Disabled ButtonStyle
		Menu     ButtonStyle
	}
	Oscilloscope  OscilloscopeStyle
	NumericUpDown NumericUpDownStyle
	DialogTitle   LabelStyle
	DialogText    LabelStyle
	SongPanel     struct {
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
		Play       color.NRGBA
		OneBeat    color.NRGBA
		TwoBeat    color.NRGBA
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
		CellBg     color.NRGBA
		Play       color.NRGBA
	}
	Menu struct {
		Text     LabelStyle
		ShortCut color.NRGBA
		Hover    color.NRGBA
		Disabled color.NRGBA
	}
	InstrumentEditor struct {
		Octave            LabelStyle
		Voices            LabelStyle
		InstrumentComment EditorStyle
		UnitComment       EditorStyle
		InstrumentList    struct {
			Number    LabelStyle
			Name      EditorStyle
			NameMuted EditorStyle
			ScrollBar ScrollBarStyle
		}
		UnitList struct {
			Name         EditorStyle
			NameDisabled EditorStyle
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
		InvalidParam  color.NRGBA
		SendTarget    color.NRGBA
	}
	Cursor    CursorStyle
	Selection CursorStyle
	Tooltip   struct {
		Color color.NRGBA
		Bg    color.NRGBA
	}
	Popup struct {
		Bg     color.NRGBA
		Shadow color.NRGBA
	}
	ScrollBar ScrollBarStyle
}

type CursorStyle struct {
	Active    color.NRGBA
	ActiveAlt color.NRGBA // alternative color for the cursor, used e.g. when the midi input is active
	Inactive  color.NRGBA
}

//go:embed theme.yml
var defaultTheme []byte

// NewTheme returns a new theme and potentially a warning if the theme file was not found or could not be read
func NewTheme() (*Theme, error) {
	var ret Theme
	warn := ReadConfig(defaultTheme, "theme.yml", &ret)
	ret.Material.Shaper = &text.Shaper{}
	ret.Material.Icon.CheckBoxChecked = must(widget.NewIcon(icons.ToggleCheckBox))
	ret.Material.Icon.CheckBoxUnchecked = must(widget.NewIcon(icons.ToggleCheckBoxOutlineBlank))
	ret.Material.Icon.RadioChecked = must(widget.NewIcon(icons.ToggleRadioButtonChecked))
	ret.Material.Icon.RadioUnchecked = must(widget.NewIcon(icons.ToggleRadioButtonUnchecked))
	return &ret, warn
}

func must[T any](ic T, err error) T {
	if err != nil {
		panic(err)
	}
	return ic
}
