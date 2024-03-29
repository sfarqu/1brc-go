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
	"sync"
)

const FILE = "data/measurements.txt"

func main() {
	ProcessFile()
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
	data sync.Map
}

func (m *StatsMap) Set(new Measurement) {
	s, _ := m.data.LoadOrStore(new.Station, NewStationStats())
	stats := s.(StationStats)
	stats.Min = math.Min(stats.Min, new.Value)
	stats.Max = math.Max(stats.Max, new.Value)
	stats.Sum = stats.Sum + new.Value
	stats.Count = stats.Count + 1
	m.data.Store(new.Station, stats)
}

// v2: use sync pools for allocating buffers and strings that get created in loop
// create gofunc to process each buffer read concurrently, and lines within the buffer
// use sync map to prevent thread conflicts
func ProcessFile() {
	stats := StatsMap{}
	f, err := os.Open(FILE)
	if err != nil {
		fmt.Println("cannot open file", err)

	}
	defer f.Close()

	bufferPool := sync.Pool{New: func() interface{} {
		buffer := make([]byte, 512*1024)
		return buffer
	}}
	var wg sync.WaitGroup

	// read lines from file using buffered reader
	r := bufio.NewReader(f)
	for {
		// 4k buffer randomly because that was in the example I copy-pasted, but it works well because it is
		// the same size as the internal buffer of r
		// this code will create many buffers inside the loop which isn't great but that what iteration is for
		buf := bufferPool.Get().([]byte)
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
		n = len(remainderOfLine)

		if n > 0 && err != io.EOF {
			remainderOfLine = remainderOfLine[:n-1]
			buf = append(buf, remainderOfLine...)
		}

		wg.Add(1)
		go func() {

			data := string(buf)
			bufferPool.Put(buf)
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
			wg.Done()
		}()

	}
	wg.Wait()

	// sort keys
	keys := make([]string, 0, 10000)

	stats.data.Range(func(key interface{}, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)

	// print in order
	var buffer bytes.Buffer
	for _, k := range keys {
		v, _ := stats.data.Load(k)
		buffer.WriteString(fmt.Sprintf("%s: %v ", k, v))
	}
	fmt.Println(buffer.String())
}
