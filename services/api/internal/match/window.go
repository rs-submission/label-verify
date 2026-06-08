package match

import (
	"sort"
	"strings"
)

type TextRegion struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	W          float64 `json:"w"`
	H          float64 `json:"h"`
}

type TextCandidate struct {
	Text       string
	Confidence float64
	X          float64
	Y          float64
	W          float64
	H          float64
	Regions    int
	Document   bool
}

const (
	defaultMaxWindowRegions = 8
	lineYTolerance          = 0.03
)

func AssembleWindows(pool []TextRegion, maxGap float64) []string {
	candidates := AssembleCandidateWindows(pool, maxGap)
	windows := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		windows = append(windows, candidate.Text)
	}
	return windows
}

func AssembleCandidateWindows(pool []TextRegion, maxGap float64) []TextCandidate {
	if len(pool) == 0 {
		return nil
	}

	regions := append([]TextRegion(nil), pool...)
	lineTolerance := adaptiveLineTolerance(regions)
	gapTolerance := adaptiveGapTolerance(regions, maxGap)
	sort.SliceStable(regions, func(i, j int) bool {
		if abs(regions[i].Y-regions[j].Y) <= lineTolerance {
			return regions[i].X < regions[j].X
		}
		return regions[i].Y < regions[j].Y
	})

	var windows []TextCandidate
	for start := range regions {
		candidate := emptyCandidate()
		for end := start; end < len(regions) && end < start+defaultMaxWindowRegions; end++ {
			if end > start && !isAdjacent(regions[end-1], regions[end], lineTolerance, gapTolerance) {
				break
			}
			candidate = candidate.with(regions[end])
			windows = append(windows, candidate)
		}
	}
	return windows
}

func isAdjacent(prev, next TextRegion, lineTolerance, maxGap float64) bool {
	if maxGap <= 0 {
		maxGap = 0.25
	}
	if abs(prev.Y-next.Y) <= lineTolerance {
		return next.X-(prev.X+prev.W) <= maxGap
	}
	return next.Y-(prev.Y+prev.H) <= maxGap
}

func emptyCandidate() TextCandidate {
	return TextCandidate{
		Confidence: 1,
		X:          0,
		Y:          0,
		W:          0,
		H:          0,
	}
}

func (c TextCandidate) with(region TextRegion) TextCandidate {
	if c.Regions == 0 {
		return TextCandidate{
			Text:       strings.TrimSpace(region.Text),
			Confidence: region.Confidence,
			X:          region.X,
			Y:          region.Y,
			W:          region.W,
			H:          region.H,
			Regions:    1,
		}
	}

	x0 := min(c.X, region.X)
	y0 := min(c.Y, region.Y)
	x1 := max(c.X+c.W, region.X+region.W)
	y1 := max(c.Y+c.H, region.Y+region.H)
	text := strings.TrimSpace(strings.Join([]string{c.Text, region.Text}, " "))
	confidence := (c.Confidence*float64(c.Regions) + region.Confidence) / float64(c.Regions+1)
	return TextCandidate{
		Text:       text,
		Confidence: confidence,
		X:          x0,
		Y:          y0,
		W:          x1 - x0,
		H:          y1 - y0,
		Regions:    c.Regions + 1,
	}
}

func adaptiveLineTolerance(regions []TextRegion) float64 {
	medianH := medianDimension(regions, func(r TextRegion) float64 { return r.H })
	if looksPixelBased(regions) {
		return max(3, medianH*0.75)
	}
	return max(lineYTolerance, medianH*0.75)
}

func adaptiveGapTolerance(regions []TextRegion, fallback float64) float64 {
	if fallback <= 0 {
		fallback = 0.25
	}
	if looksPixelBased(regions) {
		medianH := medianDimension(regions, func(r TextRegion) float64 { return r.H })
		return max(8, medianH*3)
	}
	return fallback
}

func looksPixelBased(regions []TextRegion) bool {
	for _, region := range regions {
		if region.X > 2 || region.Y > 2 || region.W > 2 || region.H > 2 {
			return true
		}
	}
	return false
}

func medianDimension(regions []TextRegion, pick func(TextRegion) float64) float64 {
	values := make([]float64, 0, len(regions))
	for _, region := range regions {
		value := pick(region)
		if value > 0 {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
