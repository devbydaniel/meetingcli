package output

import (
	"fmt"
	"io"
	"time"
)

type Formatter struct {
	w io.Writer
}

func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

func (f *Formatter) RecordingStopped(duration time.Duration) {
	fmt.Fprintf(f.w, "⏹️  Recording stopped (%s)\n", formatDuration(duration))
}

func (f *Formatter) Transcribing() {
	fmt.Fprintf(f.w, "📝 Transcribing audio...\n")
}

func (f *Formatter) TranscribeDone(path string) {
	fmt.Fprintf(f.w, "✅ Transcript saved: %s\n", path)
}

func (f *Formatter) Summarizing() {
	fmt.Fprintf(f.w, "🤖 Generating summary...\n")
}

func (f *Formatter) SummarizeDone(path string) {
	fmt.Fprintf(f.w, "✅ Summary saved: %s\n", path)
}

func (f *Formatter) MeetingComplete(dir string) {
	fmt.Fprintf(f.w, "\n📁 Meeting saved: %s\n", dir)
}

func (f *Formatter) Error(msg string) {
	fmt.Fprintf(f.w, "❌ %s\n", msg)
}

func (f *Formatter) Info(msg string) {
	fmt.Fprintf(f.w, "ℹ️  %s\n", msg)
}

func (f *Formatter) Success(msg string) {
	fmt.Fprintf(f.w, "✅ %s\n", msg)
}

func (f *Formatter) Warning(msg string) {
	fmt.Fprintf(f.w, "⚠️  %s\n", msg)
}

func (f *Formatter) MeetingListHeader() {
	fmt.Fprintf(f.w, "📁 Meetings:\n\n")
}

func (f *Formatter) MeetingListItem(name string, hasTranscript, hasSummary bool) {
	status := ""
	if hasTranscript && hasSummary {
		status = " ✅"
	} else if hasTranscript {
		status = " 📝"
	}
	fmt.Fprintf(f.w, "  %s%s\n", name, status)
}

func (f *Formatter) SetupCheck(name string, ok bool, detail string) {
	if ok {
		fmt.Fprintf(f.w, "  ✅ %s: %s\n", name, detail)
	} else {
		fmt.Fprintf(f.w, "  ❌ %s: %s\n", name, detail)
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
