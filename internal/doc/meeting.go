package doc

import (
	"strconv"
	"time"
)

func MeetingMeta(state string, started int64, duration int) string {
	switch state {
	case "recording":
		return "Recording"
	case "transcribing":
		return "Transcribing"
	case "summarizing":
		return "Summarizing"
	}
	if started <= 0 {
		return ""
	}
	out := time.UnixMilli(started).UTC().Format("2006-01-02")
	if duration > 0 {
		out += " · " + strconv.Itoa(max(1, (duration+30)/60)) + " min"
	}
	return out
}
