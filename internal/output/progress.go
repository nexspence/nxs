package output

import (
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// NewProgress returns a progressbar writing to stderr, or nil in json/plain mode.
func NewProgress(total int64, description string, jsonMode, plainMode bool) *progressbar.ProgressBar {
	if jsonMode || plainMode {
		return nil
	}
	return progressbar.NewOptions64(total,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowBytes(true),
		progressbar.OptionThrottle(50*time.Millisecond),
	)
}

// WrapReader wraps r with a progress bar. If bar is nil, returns r unchanged.
func WrapReader(r io.Reader, bar *progressbar.ProgressBar) io.Reader {
	if bar == nil {
		return r
	}
	return io.TeeReader(r, bar)
}
