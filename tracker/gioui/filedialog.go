package gioui

import (
	"image"
	"io/ioutil"
	"os"
	"path/filepath"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type FileDialog struct {
	Visible       bool
	Directory     widget.Editor
	FileList      layout.List
	FileName      widget.Editor
	BtnFolderUp   widget.Clickable
	BtnOk         widget.Clickable
	BtnCancel     widget.Clickable
	UseAltExt     widget.Bool
	ScrollBar     ScrollBar
	selectedFiles []string
	tags          []bool
}

type FileDialogStyle struct {
	dialog         *FileDialog
	save           bool
	Title          string
	DirEditorStyle material.EditorStyle
	FileNameStyle  material.EditorStyle
	FolderUpStyle  material.IconButtonStyle
	OkStyle        material.ButtonStyle
	CancelStyle    material.ButtonStyle
	UseAltExtStyle material.SwitchStyle
	ExtMain        string
	ExtAlt         string
}

func NewFileDialog() *FileDialog {
	ret := &FileDialog{
		Directory: widget.Editor{SingleLine: true, Submit: true},
		FileName:  widget.Editor{SingleLine: true, Submit: true},
		FileList:  layout.List{Axis: layout.Vertical},
		ScrollBar: ScrollBar{Axis: layout.Vertical},
	}
	wd, _ := os.Getwd()
	ret.Directory.SetText(wd)
	return ret
}

func SaveFileDialog(th *material.Theme, f *FileDialog) FileDialogStyle {
	ret := commonFileDialog(th, f)
	ret.save = true
	ret.Title = "Save As"
	ret.OkStyle = material.Button(th, &f.BtnOk, "Save")
	return ret
}

func OpenFileDialog(th *material.Theme, f *FileDialog) FileDialogStyle {
	ret := commonFileDialog(th, f)
	ret.OkStyle = material.Button(th, &f.BtnOk, "Open")
	ret.Title = "Open File"
	return ret
}

func commonFileDialog(th *material.Theme, f *FileDialog) FileDialogStyle {
	ret := FileDialogStyle{
		dialog:         f,
		FolderUpStyle:  IconButton(th, &f.BtnFolderUp, icons.NavigationArrowUpward, true),
		DirEditorStyle: material.Editor(th, &f.Directory, "Directory"),
		FileNameStyle:  material.Editor(th, &f.FileName, "Filename"),
		CancelStyle:    LowEmphasisButton(th, &f.BtnCancel, "Cancel"),
		UseAltExtStyle: material.Switch(th, &f.UseAltExt),
	}
	ret.UseAltExtStyle.Color.Enabled = white
	ret.UseAltExtStyle.Color.Disabled = white
	ret.ExtMain = ".yml"
	ret.ExtAlt = ".json"
	return ret
}

func (d *FileDialog) FileSelected() (bool, string) {
	if len(d.selectedFiles) > 0 {
		var filePath string
		filePath, d.selectedFiles = d.selectedFiles[0], d.selectedFiles[1:]
		return true, filePath
	}
	return false, ""
}

func (f *FileDialogStyle) Layout(gtx C) D {
	if f.dialog.Visible {
		for f.dialog.BtnCancel.Clicked() {
			f.dialog.Visible = false
		}
		if n := f.dialog.FileName.Text(); len(n) > 0 {
			for f.dialog.UseAltExt.Changed() {
				var extension = filepath.Ext(n)
				n = n[0 : len(n)-len(extension)]
				switch f.dialog.UseAltExt.Value {
				case true:
					n += ".json"
				default:
					n += ".yml"
				}
				f.dialog.FileName.SetText(n)
			}
		}
		fullFile := filepath.Join(f.dialog.Directory.Text(), f.dialog.FileName.Text())
		if _, err := os.Stat(fullFile); (f.save || !os.IsNotExist(err)) && f.dialog.FileName.Text() != "" {
			for f.dialog.BtnOk.Clicked() {
				f.dialog.selectedFiles = append(f.dialog.selectedFiles, fullFile)
				f.dialog.Visible = false
			}
			f.OkStyle.Color = black
			f.OkStyle.Background = primaryColor
		} else {
			f.OkStyle.Color = mediumEmphasisTextColor
			f.OkStyle.Background = inactiveSelectionColor
		}
		parent := filepath.Dir(f.dialog.Directory.Text())
		info, err := os.Stat(parent)
		if err == nil && info.IsDir() && parent != "." {
			for f.dialog.BtnFolderUp.Clicked() {
				f.dialog.Directory.SetText(parent)
			}
		} else {
			f.FolderUpStyle.Color = disabledTextColor
		}

		var subDirs, files []string
		dirList, err := ioutil.ReadDir(f.dialog.Directory.Text())
		if err == nil {
			for _, file := range dirList {
				if file.IsDir() {
					subDirs = append(subDirs, file.Name())
				} else {
					if f.dialog.UseAltExt.Value && filepath.Ext(file.Name()) == f.ExtAlt {
						files = append(files, file.Name())
					} else if !f.dialog.UseAltExt.Value && filepath.Ext(file.Name()) == f.ExtMain {
						files = append(files, file.Name())
					}
				}
			}
		}
		listLen := len(subDirs) + len(files)
		listElement := func(gtx C, index int) D {
			for len(f.dialog.tags) <= index {
				f.dialog.tags = append(f.dialog.tags, false)
			}
			for _, ev := range gtx.Events(&f.dialog.tags[index]) {
				e, ok := ev.(pointer.Event)
				if !ok {
					continue
				}
				switch e.Type {
				case pointer.Press:
					if index < len(subDirs) {
						f.dialog.Directory.SetText(filepath.Join(f.dialog.Directory.Text(), subDirs[index]))
					} else {
						f.dialog.FileName.SetText(files[index-len(subDirs)])
					}
				}
			}

			var icon *widget.Icon
			var text string
			if index < len(subDirs) {
				icon = widgetForIcon(icons.FileFolder)
				icon.Color = primaryColor
				text = subDirs[index]
			} else {
				icon = widgetForIcon(icons.EditorInsertDriveFile)
				icon.Color = primaryColor
				text = files[index-len(subDirs)]
			}
			labelColor := highEmphasisTextColor
			if text == f.dialog.FileName.Text() {
				labelColor = white
			}
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return icon.Layout(gtx, unit.Dp(24))
						}),
						layout.Rigid(Label(text, labelColor)),
					)
				}),
				layout.Expanded(func(gtx C) D {
					rect := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
					pointer.Rect(rect).Add(gtx.Ops)
					pointer.InputOp{Tag: &f.dialog.tags[index],
						Types: pointer.Press | pointer.Drag | pointer.Release,
					}.Add(gtx.Ops)
					return D{}
				}),
			)

		}
		paint.Fill(gtx.Ops, dialogBgColor)
		return layout.Center.Layout(gtx, func(gtx C) D {
			return Popup(&f.dialog.Visible).Layout(gtx, func(gtx C) D {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(Label(f.Title, white)),
						layout.Rigid(func(gtx C) D {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(f.FolderUpStyle.Layout),
								layout.Rigid(func(gtx C) D {
									return D{Size: image.Pt(gtx.Px(unit.Dp(6)), gtx.Px(unit.Dp(36)))}
								}),
								layout.Rigid(f.DirEditorStyle.Layout))
						}),
						layout.Rigid(func(gtx C) D {
							return layout.Stack{Alignment: layout.NE}.Layout(gtx,
								layout.Stacked(func(gtx C) D {
									gtx.Constraints = layout.Exact(image.Pt(gtx.Px(unit.Dp(600)), gtx.Px(unit.Dp(400))))
									if listLen > 0 {
										return f.dialog.FileList.Layout(gtx, listLen, listElement)
									} else {
										return D{Size: gtx.Constraints.Min}
									}
								}),
								layout.Expanded(func(gtx C) D {
									return f.dialog.ScrollBar.Layout(gtx, unit.Dp(10), listLen, &f.dialog.FileList.Position)
								}),
							)
						}),
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.Y = gtx.Px(unit.Dp(36))
							return layout.W.Layout(gtx, f.FileNameStyle.Layout)
						}),
						layout.Rigid(func(gtx C) D {
							gtx.Constraints = layout.Exact(image.Pt(gtx.Px(unit.Dp(600)), gtx.Px(unit.Dp(36))))
							if f.ExtAlt != "" {
								mainLabelColor := disabledTextColor
								altLabelColor := disabledTextColor
								if f.UseAltExtStyle.Switch.Value {
									altLabelColor = white
								} else {
									mainLabelColor = white
								}
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(Label(f.ExtMain, mainLabelColor)),
									layout.Rigid(f.UseAltExtStyle.Layout),
									layout.Rigid(Label(f.ExtAlt, altLabelColor)),
									layout.Flexed(1, func(gtx C) D {
										return D{Size: image.Pt(100, 1)}
									}),
									layout.Rigid(f.OkStyle.Layout),
									layout.Rigid(f.CancelStyle.Layout),
								)
							} else {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx C) D {
										return D{Size: image.Pt(100, 1)}
									}),
									layout.Rigid(f.OkStyle.Layout),
									layout.Rigid(f.CancelStyle.Layout),
								)
							}
						}),
					)
				})
			})
		})
	}
	return D{}
}
