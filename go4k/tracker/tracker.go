package tracker

import (
	"gioui.org/widget"
	"github.com/vsariola/sointu/go4k"
)

type Tracker struct {
	QuitButton     *widget.Clickable
	song           go4k.Song
	CursorRow      int
	CursorColumn   int
	DisplayPattern int
	ActiveTrack    int
}

func New() *Tracker {
	return &Tracker{
		QuitButton: new(widget.Clickable),
		song:       defaultSong,
	}
}
