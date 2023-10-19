package tracker

import "github.com/vsariola/sointu"

type (
	// ScoreRow identifies a row of the song score.
	ScoreRow struct {
		Pattern int
		Row     int
	}

	// ScorePoint identifies a row and a track in a song score.
	ScorePoint struct {
		Track int
		ScoreRow
	}

	// ScoreRect identifies a rectangular area in a song score.
	ScoreRect struct {
		Corner1 ScorePoint
		Corner2 ScorePoint
	}
)

func (r ScoreRow) AddRows(rows int) ScoreRow {
	return ScoreRow{Row: r.Row + rows, Pattern: r.Pattern}
}

func (r ScoreRow) AddPatterns(patterns int) ScoreRow {
	return ScoreRow{Row: r.Row, Pattern: r.Pattern + patterns}
}

func (r ScoreRow) Wrap(score sointu.Score) ScoreRow {
	totalRow := r.Pattern*score.RowsPerPattern + r.Row
	r.Row = mod(totalRow, score.RowsPerPattern)
	r.Pattern = mod((totalRow-r.Row)/score.RowsPerPattern, score.Length)
	return r
}

func (r ScoreRow) Clamp(score sointu.Score) ScoreRow {
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

func (r ScorePoint) AddRows(rows int) ScorePoint {
	return ScorePoint{Track: r.Track, ScoreRow: r.ScoreRow.AddRows(rows)}
}

func (r ScorePoint) AddPatterns(patterns int) ScorePoint {
	return ScorePoint{Track: r.Track, ScoreRow: r.ScoreRow.AddPatterns(patterns)}
}

func (p ScorePoint) Wrap(score sointu.Score) ScorePoint {
	p.Track = mod(p.Track, len(score.Tracks))
	p.ScoreRow = p.ScoreRow.Wrap(score)
	return p
}

func (p ScorePoint) Clamp(score sointu.Score) ScorePoint {
	if p.Track < 0 {
		p.Track = 0
	} else if l := len(score.Tracks); p.Track >= l {
		p.Track = l - 1
	}
	p.ScoreRow = p.ScoreRow.Clamp(score)
	return p
}

func (r *ScoreRect) Contains(p ScorePoint) bool {
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
