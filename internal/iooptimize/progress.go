package iooptimize

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// Progress provides dual-output logging: STDOUT (user) + slog/STDERR (developer).

// Info logs to both STDOUT and slog.
func Info(msg string, args ...any) {
	fmt.Println(msg)
	slog.Info(msg, args...)
}

// Complete logs step completion.
func Complete(msg string, args ...any) {
	fmt.Println(msg)
	slog.Info(msg, args...)
}

// ProgressBar for STDERR progress updates.
type ProgressBar struct {
	label     string
	startTime int64
	lastWidth int
}

// NewProgressBar creates a progress bar.
func NewProgressBar(label string) *ProgressBar {
	return &ProgressBar{
		label:     label,
		startTime: time.Now().UnixNano(),
	}
}

// UpdateCount updates progress bar with count and speed.
func (pb *ProgressBar) UpdateCount(count int) {
	timeSpent := float64(time.Now().UnixNano()-pb.startTime) / 1_000_000_000
	speed := int64(0)
	if timeSpent > 0 {
		speed = int64(float64(count) / timeSpent)
	}

	msg := fmt.Sprintf("%s: %s - %s items/sec",
		pb.label,
		humanize.Comma(int64(count)),
		humanize.Comma(speed),
	)

	if pb.lastWidth > len(msg) {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", pb.lastWidth))
	}

	fmt.Fprintf(os.Stderr, "\r%s", msg)
	pb.lastWidth = len(msg)
}

// Clear clears the progress bar.
func (pb *ProgressBar) Clear() {
	if pb.lastWidth > 0 {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", pb.lastWidth))
		pb.lastWidth = 0
	}
}
