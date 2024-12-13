package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	enterAltScreen      = "\033[?1049h"
	exitAltScreen       = "\033[?1049l"
	hideCursor          = "\033[?25l"
	showCursor          = "\033[?25h"
	clearScreenSequence = "\033[2J\033[H\033[3J"
	defaultTermHeight   = 24
	colorReset          = "\033[0m"
	colorRed            = "\033[31m"
	colorGreen          = "\033[32m"
	colorCyan           = "\033[36m"
	colorGrey           = "\033[90m"
	fps                 = 240
	frameTime           = time.Second / time.Duration(fps)
	bufferSize          = 4096
)

func setupTerminalCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print(showCursor + exitAltScreen)
		keyboard.Close()
		os.Exit(1)
	}()
}

func initTerminal() {
	fmt.Print(enterAltScreen + hideCursor)
}

func getTerminalSize() (int, int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 80, defaultTermHeight
	}

	var rows, cols int
	fmt.Sscanf(string(out), "%d %d", &rows, &cols)
	return cols, rows
}

func printWelcomeMessage(buf *bytes.Buffer, configCount int) int {
	messages := []string{
		"Welcome!",
		"This program can run any pre-config BAT file.",
		"Author: ANKDDEV https://github.com/ankddev",
		fmt.Sprintf("Version: %s", version),
		"===",
		fmt.Sprintf("Found %d pre-configs", configCount),
		"\nUsing ARROWS on your keyboard, select BAT file from list for running or select 'Run BLOCKCHECK (Auto-setting BAT parameters)' or select 'Exit'.\n",
		"For selection press ENTER.",
	}

	for _, msg := range messages {
		buf.WriteString(msg + "\n")
	}
	return len(messages)
}

var version string

func getOptions() ([]string, int) {
	options := []string{
		"Exit",
		"Run BLOCKCHECK (Auto-setting BAT parameters)",
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return options, 0
	}

	files, err := os.ReadDir(filepath.Join(currentDir, "pre-configs"))
	if err != nil {
		return options, 0
	}

	var batFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".bat") {
			batFiles = append(batFiles, f.Name())
		}
	}

	sort.Strings(batFiles)
	options = append(options, batFiles...)

	return options, len(batFiles)
}

func Run() error {
	setupTerminalCleanup()
	defer fmt.Print(showCursor + exitAltScreen)

	initTerminal()

	_, termHeight := getTerminalSize()

	var buf bytes.Buffer

	options, configCount := getOptions()
	if len(options) == 0 {
		fmt.Println("Can't find any BAT files in current directory.")
		return nil
	}

	currentLine := printWelcomeMessage(&buf, configCount)
	fmt.Print(buf.String())

	if err := runMainLoop(&buf, options, currentLine, termHeight); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runMainLoop(buf *bytes.Buffer, options []string, startRow, termHeight int) error {
	buf.Grow(bufferSize)

	output := bufio.NewWriter(os.Stdout)
	defer output.Flush()

	currentSelection := 0
	scrollOffset := 0
	maxVisibleOptions := min(15, termHeight-startRow-3)

	if err := keyboard.Open(); err != nil {
		return fmt.Errorf("error initializing keyboard: %v", err)
	}
	defer keyboard.Close()

	for {
		start := time.Now()

		buf.Reset()
		buf.WriteString("\033[H\033[J")

		options, configCount := getOptions()
		printWelcomeMessage(buf, configCount)

		if scrollOffset > 0 {
			buf.WriteString(fmt.Sprintf("%s↑ more items above%s\n", colorGrey, colorReset))
		}

		if currentSelection >= len(options) {
			currentSelection = len(options) - 1
		}

		endIdx := min(scrollOffset+maxVisibleOptions, len(options))

		if currentSelection >= scrollOffset+maxVisibleOptions-1 {
			scrollOffset = currentSelection - maxVisibleOptions + 2
		} else if currentSelection < scrollOffset {
			scrollOffset = currentSelection
		}

		if scrollOffset < 0 {
			scrollOffset = 0
		}
		if scrollOffset > len(options)-maxVisibleOptions {
			scrollOffset = max(0, len(options)-maxVisibleOptions)
		}

		for i := scrollOffset; i < endIdx; i++ {
			if i == currentSelection {
				buf.WriteString(fmt.Sprintf("%s► %s%s\n", colorCyan, options[i], colorReset))
			} else {
				buf.WriteString(fmt.Sprintf("  %s\n", options[i]))
			}
		}

		if endIdx < len(options) {
			buf.WriteString(fmt.Sprintf("%s↓ more items below%s\n", colorGrey, colorReset))
		}

		output.Write(buf.Bytes())
		output.Flush()

		elapsed := time.Since(start)
		if elapsed < frameTime {
			time.Sleep(frameTime - elapsed)
		}

		if _, key, err := keyboard.GetKey(); err == nil {
			switch key {
			case keyboard.KeyArrowUp:
				if currentSelection > 0 {
					currentSelection--
					if currentSelection < scrollOffset {
						scrollOffset = currentSelection
					}
				}
			case keyboard.KeyArrowDown:
				if currentSelection < len(options)-1 {
					currentSelection++
					if currentSelection >= scrollOffset+maxVisibleOptions {
						scrollOffset = currentSelection - maxVisibleOptions + 1
					}
				}
			case keyboard.KeyEnter:
				return handleSelection(options[currentSelection])
			case keyboard.KeyEsc:
				return nil
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleSelection(selected string) error {
	switch selected {
	case "Run BLOCKCHECK (Auto-setting BAT parameters)":
		fmt.Print(showCursor + exitAltScreen)
		keyboard.Close()

		_, err := runPowershellCommand("Start-Process 'blockcheck.cmd'")
		if err != nil {
			return fmt.Errorf("%s⚠ Error running BLOCKCHECK: %v%s", colorRed, err, colorReset)
		}
		return nil
	case "Exit":
		return nil
	default:
		fmt.Print(showCursor + exitAltScreen)
		keyboard.Close()

		currentDir, err := os.Getwd()
		if err != nil {
			return err
		}
		batPath := filepath.Join(currentDir, "pre-configs", selected)

		cmd := exec.Command("cmd", "/c", batPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s⚠ Error running BAT file: %v%s", colorRed, err, colorReset)
		}
		return nil
	}
}

func runPowershellCommand(command string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PowerShell error: %v - %s", err, string(output))
	}
	return string(output), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
