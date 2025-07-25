package ui

import (
	"fmt"
	"io"
	"time"

	"github.com/theckman/yacspin"
)

type consoleUI struct {
	w   io.Writer
	cfg config
}

func (c *consoleUI) Info(fmtStr string, args ...interface{}) {
	fmt.Fprintf(c.w, "[ℹ] "+fmtStr+"\n", args...)
}

func (c *consoleUI) Warn(fmtStr string, args ...interface{}) {
	fmt.Fprintf(c.w, "[!] "+fmtStr+"\n", args...)
}

func (c *consoleUI) Error(fmtStr string, args ...interface{}) {
	fmt.Fprintf(c.w, "[✗] "+fmtStr+"\n", args...)
}

func (c *consoleUI) StartSpinner(msg string) (Spinner, error) {
	if !c.cfg.enableSpinner {
		c.Info(msg + " …")
		return nopSpinner{}, nil
	}
	s, err := yacspin.New(yacspin.Config{
		Frequency: 100 * time.Millisecond,
		Writer:    c.w,
		Message:   msg + " ",
	})
	if err != nil {
		return nil, err
	}
	_ = s.Start()
	return s, nil
}

func (c *consoleUI) Progress(current, total int64) {
	pct := float64(current) / float64(total) * 100
	fmt.Fprintf(c.w, "\r  → %d/%d (%.1f%%)", current, total, pct)
	if current >= total {
		fmt.Fprint(c.w, "\n")
	}
}

type nopSpinner struct{}

func (nopSpinner) Update(msg string) error { return nil }
func (nopSpinner) Stop() error             { return nil }
