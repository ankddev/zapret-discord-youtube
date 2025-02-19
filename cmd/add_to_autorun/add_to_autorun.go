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
	fps                 = 240
	frameTime           = time.Second / time.Duration(fps)
	bufferSize          = 4096
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

func getOptions() ([]string, int) {
	options := []string{
		"Exit",
		"Delete service from autorun",
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

func clearScreen(buf *bytes.Buffer) {
	buf.WriteString(clearScreenSequence)
}

func printWelcomeMessage(buf *bytes.Buffer, configCount int) {
	messages := []string{
		"Welcome!",
		"This program can install BAT file as service with autorun.",
		"Author: ANKDDEV https://github.com/ankddev",
		fmt.Sprintf("Version: %s", version),
		"===",
		fmt.Sprintf("Found %d pre-configs", configCount),
		"\nUsing ARROWS on your keyboard, select BAT file from list for installing service 'discordfix_zapret' or select 'Delete service from autorun' or 'Run BLOCKCHECK (Auto-setting BAT parameters)' or select 'Exit'.\n",
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
	buf.Grow(bufferSize)

	// Pre-calculate terminal dimensions
	_, termHeight := getTerminalSize()
	visibleItems := termHeight - 12
	startIdx := 0

	// Create output buffer for direct writes
	output := bufio.NewWriter(os.Stdout)
	defer output.Flush()

	options, configCount := getOptions()
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

	// Main loop
	for {
		start := time.Now()

		buf.Reset()
		buf.WriteString("\033[H\033[J")

		printWelcomeMessage(&buf, configCount)

		// Calculate visible range and scroll position
		endIdx := min(startIdx+visibleItems, len(options))

		// Update scroll position before selection becomes invisible
		if currentSelection >= startIdx+visibleItems-1 {
			startIdx = currentSelection - visibleItems + 2
		} else if currentSelection < startIdx {
			startIdx = currentSelection
		}

		// Ensure startIdx stays within bounds
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx > len(options)-visibleItems {
			startIdx = max(0, len(options)-visibleItems)
		}

		// Show scroll indicators only if needed
		if startIdx > 0 {
			buf.WriteString(fmt.Sprintf("%s↑ more items above%s\n", colorGrey, colorReset))
		}

		// Batch write visible options with proper spacing
		for i := startIdx; i < endIdx; i++ {
			if i == currentSelection {
				buf.WriteString(fmt.Sprintf("%s► %s%s\n", colorCyan, options[i], colorReset))
			} else {
				buf.WriteString(fmt.Sprintf("  %s\n", options[i]))
			}
		}

		// Show bottom scroll indicator with proper spacing
		if endIdx < len(options) {
			buf.WriteString(fmt.Sprintf("%s↓ more items below%s\n", colorGrey, colorReset))
		}

		// Single write operation to terminal
		output.Write(buf.Bytes())
		output.Flush()

		// Precise frame timing
		elapsed := time.Since(start)
		if elapsed < frameTime {
			time.Sleep(frameTime - elapsed)
		}

		// Non-blocking keyboard input
		if _, key, err := keyboard.GetKey(); err == nil {
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
				case "Exit":
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
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Add helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (sm *ServiceManager) createService() error {
	fmt.Println("=== Creating service ===")

	// Get absolute path to executable
	exePath, err := sm.getExecutablePath()
	if err != nil {
		return err
	}

	// Create service with auto-start
	_, err = sm.runPowershellCommand(fmt.Sprintf("Start-Process 'sc.exe' -ArgumentList 'create %s start= auto binPath= \"%s\"' -Verb RunAs", sm.serviceName, exePath))
	if err != nil {
		return fmt.Errorf("error creating service: %v", err)
	}
	fmt.Printf("%s✓ Service created successfully.%s\n", colorGreen, colorReset)

	// Set description
	_, err = sm.runPowershellCommand(fmt.Sprintf("Start-Process 'sc.exe' -ArgumentList 'description %s \"Service for bypassing DPI blocks\"' -Verb RunAs", sm.serviceName))
	if err != nil {
		fmt.Printf("%s⚠ Error setting service description: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Printf("%s✓ Service description set.%s\n", colorGreen, colorReset)
	}

	// Configure service recovery options
	_, err = sm.runPowershellCommand(fmt.Sprintf("Start-Process 'sc.exe' -ArgumentList 'failure %s reset= 0 actions= restart/60000' -Verb RunAs", sm.serviceName))
	if err != nil {
		fmt.Printf("%s⚠ Error setting service recovery options: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Printf("%s✓ Service recovery options set.%s\n", colorGreen, colorReset)
	}

	return nil
}

func (sm *ServiceManager) getExecutablePath() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(currentDir, "bin", "winws.exe"), nil
}
