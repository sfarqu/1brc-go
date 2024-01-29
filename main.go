package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const FILE = "data/measurements.txt"

func main() {
	naiveImplementation()
}

// Measurement is a single temperature measurement
type Measurement struct {
	Station string
	Value   float64
}

// Create a new Measurement struct from a string in the format; Baoding;38.8671
func NewMeasurement(s string) (*Measurement, error) {
	tokens := strings.Split(s, ";")
	if len(tokens) != 2 {
		return nil, errors.New(fmt.Sprintf("expected 2 tokens, %s", s))
	}
	value, err := strconv.ParseFloat(tokens[1], 64)
	if err != nil {
		return nil, err
	}
	return &Measurement{
		tokens[0],
		value,
	}, nil
}

// Statistics for a single weather station
type StationStats struct {
	Min   float64
	Max   float64
	Sum   float64
	Count int64
}

// Ensure min and max are always set such that comparisons with new values will work
func NewStationStats() StationStats {
	return StationStats{
		Min:   math.MaxFloat64,
		Max:   math.SmallestNonzeroFloat64,
		Sum:   0,
		Count: 0,
	}
}

// Implement String interface to return values in expected format
func (s StationStats) String() string {
	mn := math.Round(s.Min*10) / 10
	mx := math.Round(s.Max*10) / 10
	sum := math.Round(s.Sum*10) / 10

	return fmt.Sprintf("%.1f/%.1f/%.1f", mn, sum/float64(s.Count), mx)
}

// Map of all weather stations
type StatsMap struct {
	data map[string]StationStats
}

func (m StatsMap) Set(new Measurement) {
	stats, ok := m.data[new.Station]
	if !ok {
		stats = NewStationStats()
	}
	stats.Min = math.Min(stats.Min, new.Value)
	stats.Max = math.Max(stats.Max, new.Value)
	stats.Sum = stats.Sum + new.Value
	stats.Count = stats.Count + 1
	m.data[new.Station] = stats
}

func naiveImplementation() {
	stats := StatsMap{
		make(map[string]StationStats, 450),
	}
	f, err := os.Open(FILE)
	if err != nil {
		fmt.Println("cannot able to read the file", err)

	}
	defer f.Close()

	// read lines from file using buffered reader
	r := bufio.NewReader(f)
	for {
		// 4k buffer randomly because that was in the example I copy-pasted, but it works well because it is
		// the same size as the internal buffer of r
		// this code will create many buffers inside the loop which isn't great but that what iteration is for
		buf := make([]byte, 4*1024)
		n, err := r.Read(buf)
		// exclude the final character (for some reason?)
		buf = buf[:n]
		if n == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				break
			}
			// something unexpected has happened if we get here
			return
		}
		// buffer might have stopped in the middle of a row so get rest of line
		remainderOfLine, err := r.ReadBytes('\n')

		if err != io.EOF {
			buf = append(buf, remainderOfLine...)
		}

		data := string(buf)
		lines := strings.Split(data, "\n")

		// for each line, collate into stats for each station
		for _, line := range lines {
			if len(line) == 0 {
				break
			}
			m, err := NewMeasurement(line)
			// this is here for debugging to ensure data set is valid
			if err != nil {
				fmt.Println(err)
				break
			}
			stats.Set(*m)
		}
	}

	// sort keys
	keys := make([]string, 0, len(stats.data))

	for k := range stats.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// print in order
	var buffer bytes.Buffer
	for _, k := range keys {
		buffer.WriteString(fmt.Sprintf("%s: %v ", k, stats.data[k]))
	}
	fmt.Println(buffer.String())
}
