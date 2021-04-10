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

func (r SongRow) AddRows(rows int) SongRow {
	return SongRow{Row: r.Row + rows, Pattern: r.Pattern}
}

func (r SongRow) AddPatterns(patterns int) SongRow {
	return SongRow{Row: r.Row, Pattern: r.Pattern + patterns}
}

func (r SongRow) Wrap(score sointu.Score) SongRow {
	totalRow := r.Pattern*score.RowsPerPattern + r.Row
	r.Row = mod(totalRow, score.RowsPerPattern)
	r.Pattern = mod((totalRow-r.Row)/score.RowsPerPattern, score.Length)
	return r
}

func (r SongRow) Clamp(score sointu.Score) SongRow {
	totalRow := r.Pattern*score.RowsPerPattern + r.Row
	if totalRow < 0 {
		totalRow = 0
	}
	if totalRow >= score.LengthInRows() {
		totalRow = score.LengthInRows() - 1
	}
	r.Row = totalRow % score.RowsPerPattern
	r.Pattern = ((totalRow - r.Row) / score.RowsPerPattern) % score.Length
	return r
}

func (r SongPoint) AddRows(rows int) SongPoint {
	return SongPoint{Track: r.Track, SongRow: r.SongRow.AddRows(rows)}
}

func (r SongPoint) AddPatterns(patterns int) SongPoint {
	return SongPoint{Track: r.Track, SongRow: r.SongRow.AddPatterns(patterns)}
}

func (p SongPoint) Wrap(score sointu.Score) SongPoint {
	p.Track = mod(p.Track, len(score.Tracks))
	p.SongRow = p.SongRow.Wrap(score)
	return p
}

func (p SongPoint) Clamp(score sointu.Score) SongPoint {
	if p.Track < 0 {
		p.Track = 0
	} else if l := len(score.Tracks); p.Track >= l {
		p.Track = l - 1
	}
	p.SongRow = p.SongRow.Clamp(score)
	return p
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
	if a < 0 {
		return b - 1 - mod(-a-1, b)
	}
	return a % b
}
