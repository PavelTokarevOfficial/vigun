package processing

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/composition"
)

var srtTiming = regexp.MustCompile(`(?m)^(\d{2}:\d{2}:\d{2},\d{3}) --> (\d{2}:\d{2}:\d{2},\d{3})\s*$`)

func srtHasCues(path string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return srtTiming.Match(raw), nil
}

func applyTimelineToSRT(path string, segments []composition.Segment) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	blocks := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n")
	output := make([]string, 0, len(blocks))
	index := 1
	for _, block := range blocks {
		match := srtTiming.FindStringSubmatch(block)
		if len(match) != 3 {
			continue
		}
		start, err := parseSRTTime(match[1])
		if err != nil {
			return err
		}
		end, err := parseSRTTime(match[2])
		if err != nil {
			return err
		}
		text := strings.TrimSpace(srtTiming.ReplaceAllString(block, ""))
		lines := strings.Split(text, "\n")
		if len(lines) > 0 {
			if _, numberErr := strconv.Atoi(strings.TrimSpace(lines[0])); numberErr == nil {
				text = strings.TrimSpace(strings.Join(lines[1:], "\n"))
			}
		}
		offset := time.Duration(0)
		for _, segment := range segments {
			segmentStart := secondsDuration(segment.Start)
			segmentEnd := secondsDuration(segment.End)
			if segment.Source == "" || segment.Source == "clip" {
				visibleStart := maxDuration(start, segmentStart)
				visibleEnd := minDuration(end, segmentEnd)
				if visibleEnd > visibleStart && text != "" {
					shiftedStart := offset + visibleStart - segmentStart
					shiftedEnd := offset + visibleEnd - segmentStart
					output = append(output, fmt.Sprintf("%d\n%s --> %s\n%s", index, formatSRTTime(shiftedStart), formatSRTTime(shiftedEnd), text))
					index++
				}
			}
			offset += segmentEnd - segmentStart
		}
	}
	return os.WriteFile(path, []byte(strings.Join(output, "\n\n")+"\n"), 0o600)
}

func layerSubtitleFiles(source, dir string, layers []composition.Layer) (map[string]string, error) {
	paths := map[string]string{}
	for index, layer := range layers {
		if !layer.Visible || layer.Type != "subtitles" || (layer.StartTime == 0 && layer.EndTime == 0) {
			continue
		}
		path := fmt.Sprintf("%s/subtitles-layer-%d.srt", strings.TrimRight(dir, "/"), index)
		if err := trimSRTToOutputRange(source, path, layer.StartTime, layer.EndTime); err != nil {
			return nil, err
		}
		hasCues, err := srtHasCues(path)
		if err != nil {
			return nil, err
		}
		if hasCues {
			paths[layer.ID] = path
		} else {
			// Keep the key so FFmpeg does not fall back to the untrimmed SRT.
			paths[layer.ID] = ""
		}
	}
	return paths, nil
}

func trimSRTToOutputRange(source, destination string, rangeStart, rangeEnd float64) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	startLimit := secondsDuration(rangeStart)
	endLimit := time.Duration(1<<63 - 1)
	if rangeEnd > 0 {
		endLimit = secondsDuration(rangeEnd)
	}
	blocks := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n")
	output := make([]string, 0, len(blocks))
	index := 1
	for _, block := range blocks {
		match := srtTiming.FindStringSubmatch(block)
		if len(match) != 3 {
			continue
		}
		start, parseErr := parseSRTTime(match[1])
		if parseErr != nil {
			return parseErr
		}
		end, parseErr := parseSRTTime(match[2])
		if parseErr != nil {
			return parseErr
		}
		visibleStart := maxDuration(start, startLimit)
		visibleEnd := minDuration(end, endLimit)
		if visibleEnd <= visibleStart {
			continue
		}
		text := strings.TrimSpace(srtTiming.ReplaceAllString(block, ""))
		lines := strings.Split(text, "\n")
		if len(lines) > 0 {
			if _, numberErr := strconv.Atoi(strings.TrimSpace(lines[0])); numberErr == nil {
				text = strings.TrimSpace(strings.Join(lines[1:], "\n"))
			}
		}
		if text == "" {
			continue
		}
		output = append(output, fmt.Sprintf("%d\n%s --> %s\n%s", index, formatSRTTime(visibleStart), formatSRTTime(visibleEnd), text))
		index++
	}
	return os.WriteFile(destination, []byte(strings.Join(output, "\n\n")+"\n"), 0o600)
}

func parseSRTTime(value string) (time.Duration, error) {
	var hours, minutes, seconds, millis int
	if _, err := fmt.Sscanf(value, "%d:%d:%d,%d", &hours, &minutes, &seconds, &millis); err != nil {
		return 0, fmt.Errorf("invalid SRT timestamp %q", value)
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(millis)*time.Millisecond, nil
}

func formatSRTTime(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	totalMillis := value.Milliseconds()
	hours := totalMillis / 3_600_000
	minutes := (totalMillis / 60_000) % 60
	seconds := (totalMillis / 1_000) % 60
	millis := totalMillis % 1_000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}

func secondsDuration(value float64) time.Duration {
	return time.Duration(value * float64(time.Second))
}

func maxDuration(left, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
