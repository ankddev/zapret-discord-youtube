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
	serviceName         = "zapret_by_ankddev"
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
	fmt.Println("=== Removing existing service using nssm remove ===")
	nssmPath := filepath.Join("bin", "nssm.exe")

	// Attempt to stop the service; ignore errors.
	stopCmd := exec.Command(nssmPath, "stop", sm.serviceName)
	stopCmd.Stdout = os.Stdout
	stopCmd.Stderr = os.Stderr
	_ = stopCmd.Run()
	time.Sleep(2 * time.Second)

	// Attempt service removal.
	removeCmd := exec.Command(nssmPath, "remove", sm.serviceName, "confirm")
	removeCmd.Stdout = nil
	removeCmd.Stderr = nil
	err := removeCmd.Run()
	if err != nil {
		errMsg := err.Error()
		// If error indicates the service is not present or exit status 3, ignore it.
		if strings.Contains(errMsg, "Указанная служба не установлена") ||
			strings.Contains(strings.ToLower(errMsg), "not found") ||
			strings.Contains(errMsg, "marked for deletion") ||
			strings.Contains(errMsg, "can't open service") ||
			strings.Contains(errMsg, "exit status 3") {
			fmt.Println("Service not found or already marked for deletion; continuing...")
			err = nil
		} else {
			fmt.Printf("%s⚠ Error while removing service via nssm: %v%s\n", colorRed, err, colorReset)
			return err
		}
	} else {
		fmt.Printf("%s✓ Service removed successfully via nssm.%s\n", colorGreen, colorReset)
	}
	// Allow extra time for Windows to complete deletion.
	time.Sleep(3 * time.Second)
	return nil
}

func (sm *ServiceManager) installService(batFilePath string) error {
	// Remove the service and wait until it's fully gone.
	_ = sm.removeService()
	nssmPath := filepath.Join("bin", "nssm.exe")
	_ = exec.Command(nssmPath, "status", sm.serviceName).Run()

	fmt.Println("=== Installing new service ===")
	fmt.Printf("► Installing file as service: %s\n", batFilePath)

	_ = exec.Command(nssmPath, "install", sm.serviceName, "cmd.exe", "/c", batFilePath).Run()

	time.Sleep(3 * time.Second)

	// Configure NSSM to ignore exit code 1 to avoid SERVICE_PAUSED issues.
	setExitCmd := exec.Command(nssmPath, "set", sm.serviceName, "AppExit", "1", "Ignore")
	setExitCmd.Stdout = os.Stdout
	setExitCmd.Stderr = os.Stderr
	_ = setExitCmd.Run()

	// Set AppRestartDelay to 5000 milliseconds.
	setDelayCmd := exec.Command(nssmPath, "set", sm.serviceName, "AppRestartDelay", "5000")
	setDelayCmd.Stdout = os.Stdout
	setDelayCmd.Stderr = os.Stderr
	_ = setDelayCmd.Run()

	// Start the service.
	fmt.Println("► Starting service...")
	startCmd := exec.Command(nssmPath, "start", sm.serviceName)
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	err := startCmd.Run()
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "service_paused") {
			fmt.Println("Service is paused; attempting restart via nssm restart...")
			restartCmd := exec.Command(nssmPath, "restart", sm.serviceName)
			restartCmd.Stdout = os.Stdout
			restartCmd.Stderr = os.Stderr
			err = restartCmd.Run()
			if err != nil {
				fmt.Printf("%s⚠ Error while restarting service via nssm: %v%s\n", colorRed, err, colorReset)
				return err
			}
		} else {
			fmt.Printf("%s⚠ Error while starting service via nssm: %v%s\n", colorRed, err, colorReset)
			return err
		}
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
