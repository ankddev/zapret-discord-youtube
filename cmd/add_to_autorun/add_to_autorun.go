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
	serviceName         = "discordfix_zapret"
	clearScreenSequence = "\033[2J\033[H\033[3J"
	defaultTermHeight   = 24
	enterAltScreen      = "\033[?1049h"
	exitAltScreen       = "\033[?1049l"
	hideCursor          = "\033[?25l"
	showCursor          = "\033[?25h"
	fps                 = 144
	frameTime           = time.Second / time.Duration(fps)
)

var version string

// ServiceManager handles Windows service operations
type ServiceManager struct {
	serviceName string
}

// UI constants
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGrey   = "\033[90m"
)

func NewServiceManager(name string) *ServiceManager {
	return &ServiceManager{serviceName: name}
}

func (sm *ServiceManager) runPowershellCommand(command string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PowerShell error: %v - %s", err, string(output))
	}
	return string(output), nil
}

func (sm *ServiceManager) removeService() error {
	fmt.Println("=== Deleting existing service ===")

	// Stop service
	fmt.Printf("► Stopping service '%s'...\n", sm.serviceName)
	_, err := sm.runPowershellCommand(fmt.Sprintf("Start-Process 'sc.exe' -ArgumentList 'stop %s' -Verb RunAs", sm.serviceName))
	if err != nil {
		fmt.Printf("%s⚠ Error while stopping service: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Printf("%s✓ Service stopped successfully.%s\n", colorGreen, colorReset)
	}

	// Terminate process
	fmt.Println("► Shutting down process 'winws.exe'...")
	_, err = sm.runPowershellCommand("Start-Process 'powershell' -ArgumentList 'Stop-Process -Name \"winws\" -Force' -Verb RunAs")
	if err != nil {
		fmt.Printf("%s⚠ Error while terminating process: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Printf("%s✓ Process terminated successfully.%s\n", colorGreen, colorReset)
	}

	// Delete service
	fmt.Printf("► Deleting service '%s'...\n", sm.serviceName)
	_, err = sm.runPowershellCommand(fmt.Sprintf("Start-Process 'sc.exe' -ArgumentList 'delete %s' -Verb RunAs", sm.serviceName))
	if err != nil {
		fmt.Printf("%s⚠ Error while deleting service: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Printf("%s✓ Service deleted successfully.%s\n", colorGreen, colorReset)
	}

	return nil
}

func (sm *ServiceManager) installService(batFilePath string) error {
	// First remove existing service
	err := sm.removeService()
	if err != nil {
		return err
	}

	fmt.Println("=== Installing new service ===")
	fmt.Printf("► Installing file as service: %s\n", batFilePath)

	// Create service
	createCmd := fmt.Sprintf(
		`$process = Start-Process 'sc.exe' -ArgumentList 'create %s binPath= "cmd.exe /c \"%s\"" start= auto' -Verb RunAs -PassThru; $process.WaitForExit(); Write-Output $process.ExitCode`,
		sm.serviceName,
		batFilePath,
	)

	_, err = sm.runPowershellCommand(createCmd)
	if err != nil {
		fmt.Printf("%s⚠ Error while creating service: %v%s\n", colorRed, err, colorReset)
		return err
	}

	// Start service
	fmt.Println("► Starting service...")
	_, err = sm.runPowershellCommand(fmt.Sprintf("Start-Process 'sc.exe' -ArgumentList 'start %s' -Verb RunAs", sm.serviceName))
	if err != nil {
		fmt.Printf("%s⚠ Error while starting service: %v%s\n", colorRed, err, colorReset)
		return err
	}

	fmt.Printf("%s✓ Service started successfully.%s\n", colorGreen, colorReset)
	return nil
}

func getOptions() []string {
	options := []string{
		"Delete service from autorun",
		"Run BLOCKCHECK (Auto-setting BAT parameters)",
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return options
	}

	files, err := os.ReadDir(filepath.Join(currentDir, "pre-configs"))
	if err != nil {
		return options
	}

	var batFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".bat") {
			batFiles = append(batFiles, f.Name())
		}
	}

	sort.Strings(batFiles)
	options = append(options, batFiles...)

	return options
}

func clearScreen(buf *bytes.Buffer) {
	buf.WriteString(clearScreenSequence)
}

func printWelcomeMessage(buf *bytes.Buffer) {
	messages := []string{
		"Welcome!",
		"This program can install BAT file as service with autorun.",
		"Author: ANKDDEV https://github.com/ankddev",
		fmt.Sprintf("Version: %s", version),
		"===",
		"\nUsing ARROWS on your keyboard, select BAT file from list for installing service 'discordfix_zapret' or select 'Delete service from autorun' or 'Run BLOCKCHECK (Auto-setting BAT parameters)'.\n",
		"For selection press ENTER.",
	}

	for _, msg := range messages {
		buf.WriteString(msg + "\n")
	}
}

// Add new functions for terminal handling
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

func main() {
	// Setup cleanup on exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print(showCursor + exitAltScreen)
		keyboard.Close()
		os.Exit(1)
	}()

	fmt.Print(enterAltScreen + hideCursor)
	defer fmt.Print(showCursor + exitAltScreen)

	var buf bytes.Buffer

	// First fill the buffer
	printWelcomeMessage(&buf)
	// Then clear screen and output content
	fmt.Print(clearScreenSequence)
	fmt.Print(buf.String())

	options := getOptions()
	if len(options) == 0 {
		fmt.Println("Can't find any BAT files in current directory.")
		return
	}

	currentSelection := 0
	serviceManager := NewServiceManager(serviceName)

	// Initialize keyboard
	if err := keyboard.Open(); err != nil {
		fmt.Println("Error initializing keyboard:", err)
		return
	}
	defer keyboard.Close()

	_, termHeight := getTerminalSize()
	visibleItems := termHeight - 12 // 12 lines for welcome message and padding
	startIdx := 0

	// Main loop
	for {
		start := time.Now()

		buf.Reset()
		// Move cursor to start and clear screen to the end
		buf.WriteString("\033[H\033[J")

		printWelcomeMessage(&buf)

		// Calculate range of visible elements
		if currentSelection >= startIdx+visibleItems {
			startIdx = currentSelection - visibleItems + 1
		} else if currentSelection < startIdx {
			startIdx = currentSelection
		}

		endIdx := startIdx + visibleItems
		if endIdx > len(options) {
			endIdx = len(options)
		}

		// Show up arrow if there are hidden elements above
		if startIdx > 0 {
			buf.WriteString(fmt.Sprintf("%s↑ more items above%s\n", colorGrey, colorReset))
		}

		// Output visible options
		for i := startIdx; i < endIdx; i++ {
			prefix := "  "
			if i == currentSelection {
				prefix = "► "
				buf.WriteString(fmt.Sprintf("%s%s%s%s\n", colorCyan, prefix, options[i], colorReset))
			} else {
				buf.WriteString(fmt.Sprintf("%s%s\n", prefix, options[i]))
			}
		}

		// Show down arrow if there are hidden elements below
		if endIdx < len(options) {
			buf.WriteString(fmt.Sprintf("%s↓ more items below%s\n", colorGrey, colorReset))
		}

		// Output buffer in one operation
		os.Stdout.Write(buf.Bytes())

		// Precise timing for next frame
		elapsed := time.Since(start)
		if elapsed < frameTime {
			time.Sleep(frameTime - elapsed)
		}

		// Handle keyboard input
		_, key, err := keyboard.GetKey()
		if err != nil {
			fmt.Println("Error reading keyboard:", err)
			return
		}

		switch key {
		case keyboard.KeyArrowUp:
			if currentSelection > 0 {
				currentSelection--
			}
		case keyboard.KeyArrowDown:
			if currentSelection < len(options)-1 {
				currentSelection++
			}
		case keyboard.KeyEnter:
			clearScreen(&buf)
			switch options[currentSelection] {
			case "Delete service from autorun":
				serviceManager.removeService()
			case "Run BLOCKCHECK (Auto-setting BAT parameters)":
				// Restore normal terminal state before launch
				fmt.Print(showCursor + exitAltScreen)
				keyboard.Close()

				_, err := serviceManager.runPowershellCommand("Start-Process 'blockcheck.cmd'")
				if err != nil {
					fmt.Printf("%s⚠ Error running BLOCKCHECK: %v%s\n", colorRed, err, colorReset)
				}
				return
			default:
				batPath := filepath.Join("pre-configs", options[currentSelection])
				serviceManager.installService(batPath)
			}

			fmt.Println("Ready! You can close this window")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
			return
		case keyboard.KeyEsc:
			return
		}
	}
}
