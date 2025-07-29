package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/theckman/yacspin"
)

const (
	CompleteCharacter string = "done"    //"✓",
	StopCharacter     string = "stopped" //"⏹"
	WarnFailCharacter string = "warn"    //"⚠",
	StopFailCharacter string = "error"   //"✗",
)

type consoleUI struct {
	w   io.Writer
	cfg config
}

func (c *consoleUI) Info(fmtStr string, args ...interface{}) {
	spinner, _ := c.newSpinner()
	message := padRight(formatMsg(fmtStr, args...), c.cfg.paddingChar, c.cfg.taskColumnWidth, c.cfg.minPadding)
	spinner.StopMessage(message)
	_ = spinner.Start()
	_ = spinner.Stop()
}

func (c *consoleUI) Warn(fmtStr string, args ...interface{}) {
	spinner, _ := c.newSpinner()
	message := padRight(formatMsg(fmtStr, args...), c.cfg.paddingChar, c.cfg.taskColumnWidth, c.cfg.minPadding)
	spinner.StopFailCharacter(WarnFailCharacter)
	spinner.StopFailColors("fgHiYellow")
	spinner.StopFailMessage(message)
	_ = spinner.Start()
	_ = spinner.StopFail()
}

func (c *consoleUI) Error(fmtStr string, args ...interface{}) {
	spinner, _ := c.newSpinner()
	message := padRight(formatMsg(fmtStr, args...), c.cfg.paddingChar, c.cfg.taskColumnWidth, c.cfg.minPadding)
	spinner.StopFailMessage(message)
	_ = spinner.Start()
	_ = spinner.StopFail()
}

func (c *consoleUI) Task(fmtStr string, args ...interface{}) (Spinner, error) {
	spinner, err := c.newSpinner()
	if err != nil {
		return nil, err
	}

	// Ensure the spinner has the same message across all functions
	message := padRight(formatMsg(fmtStr, args...), c.cfg.paddingChar, c.cfg.taskColumnWidth, c.cfg.minPadding)
	messageFuncs := []func(string){spinner.Message, spinner.StopMessage, spinner.StopFailMessage}
	for _, msgFunc := range messageFuncs {
		msgFunc(message)
	}

	return &TaskCompleter{
		spinner:         spinner,
		paddingChar:     c.cfg.paddingChar,
		minPadding:      c.cfg.minPadding,
		taskColumnWidth: c.cfg.taskColumnWidth,
	}, spinner.Start()
}

func (c *consoleUI) newSpinner() (*yacspin.Spinner, error) {
	spinner, err := yacspin.New(yacspin.Config{
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[c.cfg.spinnerIndex],
		SpinnerAtEnd:      true,
		ShowCursor:        false,
		SuffixAutoColon:   true,
		StopCharacter:     CompleteCharacter,
		StopFailCharacter: StopFailCharacter,
		Colors:            []string{"fgHiCyan"},
		StopColors:        []string{"fgHiGreen"},
		StopFailColors:    []string{"fgHiRed"},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create spinner: %w", err)
	}
	return spinner, nil
}

type TaskCompleter struct {
	spinner         *yacspin.Spinner
	paddingChar     string
	minPadding      int
	taskColumnWidth int
}

func (tc *TaskCompleter) Update(fmtStr string, args ...interface{}) {
	if tc.spinner == nil {
		return
	}
	msg := formatMsg(fmtStr, args...)
	tc.spinner.Message(padRight(msg, tc.paddingChar, tc.taskColumnWidth, tc.minPadding))
}

func (tc *TaskCompleter) Complete() {
	if tc.spinner == nil {
		return
	}
	_ = tc.spinner.Stop()
}

func (tc *TaskCompleter) Stop() {
	if tc.spinner == nil {
		return
	}
	tc.spinner.StopCharacter(StopCharacter)
	_ = tc.spinner.Stop()
}

func (tc *TaskCompleter) Fail() {
	if tc.spinner == nil {
		return
	}
	_ = tc.spinner.StopFail()
}

// padRight formats the input string with padding based on the provided character and fixed padding.
func padRight(input string, paddingChar string, taskColumnWidth int, minPadding int) string {

	// Calculate dynamic padding length
	if taskColumnWidth < 0 {
		taskColumnWidth = 70 // historically used default width is 80, this gives room for the spinner and stop chars
	}

	// Calculate the total padding needed
	totalPadding := taskColumnWidth - len(input) - 1 // -1 for the trailing space

	// totalPadding can never be negative, so we ensure it is at least 0
	if totalPadding < minPadding {
		totalPadding = minPadding
	}

	// Generate the repeating padding characters
	padding := strings.Repeat(paddingChar, totalPadding)

	// Construct and return the padded string
	return fmt.Sprintf("%s%s ", input, padding)
}

func formatMsg(fmtStr string, args ...interface{}) string {
	if len(args) == 0 {
		return fmtStr
	}
	return fmt.Sprintf(fmtStr, args...)
}
