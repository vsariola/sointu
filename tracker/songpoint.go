package tracker

import "github.com/vsariola/sointu"

type SongRow struct {
	Pattern int
	Row     int
}

type SongPoint struct {
	Track int
	SongRow
}

type SongRect struct {
	Corner1 SongPoint
	Corner2 SongPoint
}

func (r *SongRow) Wrap(song sointu.Song) {
	totalRow := r.Pattern*song.RowsPerPattern + r.Row
	r.Row = mod(totalRow, song.RowsPerPattern)
	r.Pattern = mod((totalRow-r.Row)/song.RowsPerPattern, song.SequenceLength())
}

func (r *SongRow) Clamp(song sointu.Song) {
	totalRow := r.Pattern*song.RowsPerPattern + r.Row
	if totalRow < 0 {
		totalRow = 0
	}
	if totalRow >= song.TotalRows() {
		totalRow = song.TotalRows() - 1
	}
	r.Row = totalRow % song.RowsPerPattern
	r.Pattern = ((totalRow - r.Row) / song.RowsPerPattern) % song.SequenceLength()
}

func (p *SongPoint) Wrap(song sointu.Song) {
	p.Track = mod(p.Track, len(song.Tracks))
	p.SongRow.Wrap(song)
}

func (p *SongPoint) Clamp(song sointu.Song) {
	if p.Track < 0 {
		p.Track = 0
	} else if l := len(song.Tracks); p.Track >= l {
		p.Track = l - 1
	}
	p.SongRow.Clamp(song)
}

func (r *SongRect) Contains(p SongPoint) bool {
	track1, track2 := r.Corner1.Track, r.Corner2.Track
	if track2 < track1 {
		track1, track2 = track2, track1
	}
	if p.Track < track1 || p.Track > track2 {
		return false
	}
	pattern1, row1, pattern2, row2 := r.Corner1.Pattern, r.Corner1.Row, r.Corner2.Pattern, r.Corner2.Row
	if pattern2 < pattern1 || (pattern1 == pattern2 && row2 < row1) {
		pattern1, row1, pattern2, row2 = pattern2, row2, pattern1, row1
	}
	if p.Pattern < pattern1 || p.Pattern > pattern2 {
		return false
	}
	if p.Pattern == pattern1 && p.Row < row1 {
		return false
	}
	if p.Pattern == pattern2 && p.Row > row2 {
		return false
	}
	return true
}

func mod(a, b int) int {
	m := a % b
	if a < 0 && b < 0 {
		m -= b
	}
	if a < 0 && b > 0 {
		m += b
	}
	return m
}
