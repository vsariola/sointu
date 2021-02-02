package tracker

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
)

const trackRowHeight = 16
const trackWidth = 54
const patmarkWidth = 16

func (t *Tracker) layoutTrack(trackNo int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = trackWidth
		gtx.Constraints.Max.X = trackWidth
		defer op.Save(gtx.Ops).Load()
		clip.Rect{Max: gtx.Constraints.Max}.Add(gtx.Ops)
		op.Offset(f32.Pt(0, float32(gtx.Constraints.Max.Y/2)-trackRowHeight)).Add(gtx.Ops)
		// TODO: this is a time bomb; as soon as one of the patterns is not the same length as rest. Find a solution
		// to fix the pattern lengths to a constant value
		cursorSongRow := t.Cursor.Pattern*t.song.RowsPerPattern + t.Cursor.Row
		op.Offset(f32.Pt(0, (-1*trackRowHeight)*float32(cursorSongRow))).Add(gtx.Ops)
		patternRect := SongRect{
			Corner1: SongPoint{SongRow: SongRow{Pattern: t.Cursor.Pattern}, Track: t.Cursor.Track},
			Corner2: SongPoint{SongRow: SongRow{Pattern: t.SelectionCorner.Pattern}, Track: t.SelectionCorner.Track},
		}
		pointRect := SongRect{
			Corner1: t.Cursor,
			Corner2: t.SelectionCorner,
		}
		for i, s := range t.song.Tracks[trackNo].Sequence {
			if patternRect.Contains(SongPoint{Track: trackNo, SongRow: SongRow{Pattern: i}}) {
				paint.FillShape(gtx.Ops, activeTrackColor, clip.Rect{Max: image.Pt(trackWidth, trackRowHeight*t.song.RowsPerPattern)}.Op())
			}
			for j := 0; j < t.song.RowsPerPattern; j++ {
				c := t.song.Tracks[trackNo].Patterns[s][j]
				songRow := SongRow{Pattern: i, Row: j}
				songPoint := SongPoint{Track: trackNo, SongRow: songRow}
				if songRow == t.PlayPosition && t.Playing {
					paint.FillShape(gtx.Ops, trackerPlayColor, clip.Rect{Max: image.Pt(trackWidth, trackRowHeight)}.Op())
				}
				if j == 0 {
					paint.ColorOp{Color: trackerPatMarker}.Add(gtx.Ops)
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, patternIndexToString(s))
				}
				if songRow == t.Cursor.SongRow {
					paint.ColorOp{Color: trackerActiveTextColor}.Add(gtx.Ops)
				} else {
					paint.ColorOp{Color: trackerInactiveTextColor}.Add(gtx.Ops)
				}
				op.Offset(f32.Pt(patmarkWidth, 0)).Add(gtx.Ops)
				if t.TrackShowHex[trackNo] {
					var text string
					switch c {
					case 0:
						text = "--"
					case 1:
						text = ".."
					default:
						text = fmt.Sprintf("%02x", c)
					}
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, strings.ToUpper(text))
					if pointRect.Contains(songPoint) {
						for col := 0; col < 2; col++ {
							color := trackerSelectionColor
							if songPoint == t.Cursor && t.CursorColumn == col {
								color = trackerCursorColor
							}
							paint.FillShape(gtx.Ops, color, clip.Rect{Min: image.Pt(col*10, 0), Max: image.Pt(col*10+10, trackRowHeight)}.Op())
						}
					}
				} else {
					widget.Label{}.Layout(gtx, textShaper, trackerFont, trackerFontSize, valueAsNote(c))
					if pointRect.Contains(songPoint) {
						color := trackerSelectionColor
						if songPoint == t.Cursor {
							color = trackerCursorColor
						}
						paint.FillShape(gtx.Ops, color, clip.Rect{Max: image.Pt(30, trackRowHeight)}.Op())
					}
				}
				op.Offset(f32.Pt(-patmarkWidth, trackRowHeight)).Add(gtx.Ops)
			}
		}
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
