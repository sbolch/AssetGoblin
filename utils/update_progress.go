package utils

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// UpdateProgress observes copied bytes without buffering the release archive.
// Redirected output receives only the initial and final download lines.
type UpdateProgress struct {
	out               io.Writer
	total, downloaded int64
	interactive       bool
	lastRender        time.Time
	style             lipgloss.Style
}

// NewUpdateProgress starts a download display using the supplied bar style.
func NewUpdateProgress(out io.Writer, total int64, interactive bool, style lipgloss.Style) *UpdateProgress {
	p := &UpdateProgress{out: out, total: total, interactive: interactive, style: style}
	p.render(false)
	return p
}

func (p *UpdateProgress) Write(data []byte) (int, error) {
	p.downloaded += int64(len(data))
	if p.interactive && time.Since(p.lastRender) >= 150*time.Millisecond {
		p.render(false)
	}
	return len(data), nil
}

// Finish prints the final byte count and ends the terminal progress line.
func (p *UpdateProgress) Finish(success bool) {
	p.render(true)
	if !success {
		lipgloss.Fprintln(p.out, lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("Download failed."))
	}
}

func (p *UpdateProgress) render(final bool) {
	message := fmt.Sprintf("Downloading: %d bytes", p.downloaded)
	if p.total > 0 {
		fraction := min(float64(p.downloaded)/float64(p.total), 1)
		filled := int(fraction * 20)
		bar := p.style.Render(strings.Repeat("━", filled)) + strings.Repeat("─", 20-filled)
		message = fmt.Sprintf("Downloading [%s] %3.0f%% (%d/%d bytes)", bar, fraction*100, p.downloaded, p.total)
	}
	if p.interactive {
		// Cursor controls are emitted only when the destination is a terminal.
		fmt.Fprint(p.out, "\r\x1b[2K")
		lipgloss.Fprint(p.out, message)
		if final {
			fmt.Fprintln(p.out)
		}
	} else {
		lipgloss.Fprintln(p.out, message)
	}
	p.lastRender = time.Now()
}
