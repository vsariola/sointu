package gioui

import (
	_ "embed"
	"image/color"

	"gioui.org/text"
	"gioui.org/unit"
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
		Tab      struct {
			Active          ButtonStyle
			Inactive        ButtonStyle
			IndicatorHeight unit.Dp
			IndicatorColor  color.NRGBA
		}
	}
	IconButton struct {
		Enabled  IconButtonStyle
		Disabled IconButtonStyle
		Emphasis IconButtonStyle
		Error    IconButtonStyle
	}
	Oscilloscope  OscilloscopeStyle
	NumericUpDown NumericUpDownStyle
	SongPanel     struct {
		RowHeader  LabelStyle
		RowValue   LabelStyle
		Expander   LabelStyle
		Version    LabelStyle
		ErrorColor color.NRGBA
		Bg         color.NRGBA
	}
	Alert      AlertStyles
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
	Dialog      DialogStyle
	OrderEditor struct {
		TrackTitle LabelStyle
		RowTitle   LabelStyle
		Cell       LabelStyle
		Loop       color.NRGBA
		CellBg     color.NRGBA
		Play       color.NRGBA
	}
	Menu struct {
		Main   MenuStyle
		Preset MenuStyle
	}
	InstrumentEditor struct {
		Octave     LabelStyle
		Properties struct {
			Label LabelStyle
		}
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
		Presets struct {
			SearchBg  color.NRGBA
			Directory LabelStyle
			Results   struct {
				Builtin LabelStyle
				User    LabelStyle
				UserDir LabelStyle
			}
		}
	}
	UnitEditor struct {
		Name          LabelStyle
		Chooser       LabelStyle
		Hint          LabelStyle
		WireColor     color.NRGBA
		WireHint      LabelStyle
		WireHighlight color.NRGBA
		Width         unit.Dp
		Height        unit.Dp
		RackComment   LabelStyle
		UnitList      struct {
			LabelWidth unit.Dp
			Name       LabelStyle
			Disabled   LabelStyle
			Error      color.NRGBA
		}
		Error   color.NRGBA
		Divider color.NRGBA
	}
	Cursor    CursorStyle
	Selection CursorStyle
	Tooltip   struct {
		Color color.NRGBA
		Bg    color.NRGBA
	}
	Popup struct {
		Menu   PopupStyle
		Dialog PopupStyle
	}
	Split        SplitStyle
	ScrollBar    ScrollBarStyle
	Knob         KnobStyle
	DisabledKnob KnobStyle
	Switch       SwitchStyle
	SignalRail   RailStyle
	Port         PortStyle

	// iconCache is used to cache the icons created from iconvg data
	iconCache map[*byte]*widget.Icon
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
	ret.iconCache = make(map[*byte]*widget.Icon)
	return &ret, warn
}

func (th *Theme) Icon(data []byte) *widget.Icon {
	if icon, ok := th.iconCache[&data[0]]; ok {
		return icon
	}
	icon := must(widget.NewIcon(data))
	th.iconCache[&data[0]] = icon
	return icon
}

func must[T any](ic T, err error) T {
	if err != nil {
		panic(err)
	}
	return ic
}
