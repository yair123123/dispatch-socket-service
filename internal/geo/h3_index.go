package geo

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type H3Indexer struct{}

type CellCoord struct {
	Resolution int
	Q          int
	R          int
}

func NewH3Indexer() *H3Indexer { return &H3Indexer{} }

func (h *H3Indexer) CellFromLatLon(lat, lon float64, resolution int) (string, error) {
	if resolution < 0 || resolution > 15 {
		return "", fmt.Errorf("invalid h3 resolution: %d", resolution)
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return "", fmt.Errorf("invalid coordinates lat=%f lon=%f", lat, lon)
	}
	scale := math.Pow(2, float64(resolution+10))
	q := int(math.Floor((lon + 180.0) * scale / 360.0))
	r := int(math.Floor((lat + 90.0) * scale / 180.0))
	return encodeCell(CellCoord{Resolution: resolution, Q: q, R: r}), nil
}

func (h *H3Indexer) CellsByRing(originCell string, ringSize int) (map[int][]string, error) {
	if ringSize < 0 {
		return nil, fmt.Errorf("ring size must be >=0")
	}
	origin, err := decodeCell(originCell)
	if err != nil {
		return nil, err
	}

	layers := make(map[int][]string, ringSize+1)
	for dq := -ringSize; dq <= ringSize; dq++ {
		for dr := max(-ringSize, -dq-ringSize); dr <= min(ringSize, -dq+ringSize); dr++ {
			ds := -dq - dr
			dist := max(abs(dq), max(abs(dr), abs(ds)))
			cell := encodeCell(CellCoord{Resolution: origin.Resolution, Q: origin.Q + dq, R: origin.R + dr})
			layers[dist] = append(layers[dist], cell)
		}
	}
	for ring := 0; ring <= ringSize; ring++ {
		sort.Strings(layers[ring])
	}
	return layers, nil
}

func encodeCell(c CellCoord) string {
	return fmt.Sprintf("%d:%d:%d", c.Resolution, c.Q, c.R)
}

func decodeCell(cell string) (CellCoord, error) {
	parts := strings.Split(cell, ":")
	if len(parts) != 3 {
		return CellCoord{}, fmt.Errorf("invalid cell id: %s", cell)
	}
	res, err := strconv.Atoi(parts[0])
	if err != nil {
		return CellCoord{}, fmt.Errorf("invalid cell resolution: %w", err)
	}
	q, err := strconv.Atoi(parts[1])
	if err != nil {
		return CellCoord{}, fmt.Errorf("invalid cell q: %w", err)
	}
	r, err := strconv.Atoi(parts[2])
	if err != nil {
		return CellCoord{}, fmt.Errorf("invalid cell r: %w", err)
	}
	return CellCoord{Resolution: res, Q: q, R: r}, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
