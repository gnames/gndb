package iooptimize

import (
	"github.com/cheggaaa/pb/v3"
)

// newProgressBar creates a new progress bar with consistent
// settings.
func newProgressBar(
	total int,
	prefix string,
) *pb.ProgressBar {
	bar := pb.Full.Start(total)
	bar.Set("prefix", prefix)
	bar.Set(pb.CleanOnFinish, true)
	return bar
}
