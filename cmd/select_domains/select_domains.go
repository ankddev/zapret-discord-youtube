package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	// Terminal control
	enterAltScreen = "\033[?1049h"
	exitAltScreen  = "\033[?1049l"
	hideCursor     = "\033[?25l"
	showCursor     = "\033[?25h"
	clearScreen    = "\033[2J\033[H"

	// Colors
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorGrey  = "\033[90m"

	// UI constants
	visibleItems     = 15
	headerLines      = 5
	scrollAreaHeight = visibleItems + 2
	fps              = 240
	frameTime        = time.Second / time.Duration(fps)
	bufferSize       = 4096
)

type FileEntry struct {
	name      string
	selected  bool
	isControl bool
}

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

func drawScreen(buf *bytes.Buffer, entries []FileEntry, currentIndex, scrollOffset int, output *bufio.Writer) {
	buf.Reset()
	buf.WriteString("\033[H\033[J")

	// Header section
	buf.WriteString("Use ↑↓ (arrows) for navigation, SPACE or ↵ (ENTER) to select\n\n")

	// Draw control options first
	for i, entry := range entries {
		if !entry.isControl {
			continue
		}

		prefix := " "
		if i == currentIndex {
			prefix = ">"
			buf.WriteString(fmt.Sprintf("%s%s %s%s\n", colorCyan, prefix, entry.name, colorReset))
		} else {
			buf.WriteString(fmt.Sprintf("%s %s\n", prefix, entry.name))
		}
	}

	buf.WriteString("\n\n") // Extra empty line and separator

	// Get file entries
	var fileEntries []FileEntry
	for _, entry := range entries {
		if !entry.isControl {
			fileEntries = append(fileEntries, entry)
		}
	}

	totalFiles := len(fileEntries)
	visibleEnd := min(scrollOffset+visibleItems, totalFiles)

	// Clear scroll area
	for i := 0; i < scrollAreaHeight; i++ {
		buf.WriteString(strings.Repeat(" ", 80) + "\n")
	}

	// Move back to start of scroll area
	buf.WriteString("\033[" + fmt.Sprint(scrollAreaHeight) + "A")

	// Show scroll indicator
	if scrollOffset > 0 {
		buf.WriteString(fmt.Sprintf("%s↑ Scroll up for more files%s\n", colorGrey, colorReset))
	} else {
		buf.WriteString("\n")
	}

	// Draw visible file entries
	for i := scrollOffset; i < visibleEnd; i++ {
		entry := fileEntries[i]
		realIndex := getRealIndex(entries, entry.name)
		prefix := " "
		if realIndex == currentIndex {
			prefix = ">"
		}
		selected := "[ ]"
		if entry.selected {
			selected = "[+]"
		}

		line := fmt.Sprintf("%s %s %s", prefix, selected, entry.name)
		if realIndex == currentIndex {
			buf.WriteString(fmt.Sprintf("%s%s%s\n", colorCyan, line, colorReset))
		} else {
			buf.WriteString(fmt.Sprintf("%s\n", line))
		}
	}

	// Fill remaining lines with empty space
	for i := visibleEnd - scrollOffset; i < visibleItems; i++ {
		buf.WriteString("\n")
	}

	// Show bottom scroll indicator
	if visibleEnd < totalFiles {
		buf.WriteString(fmt.Sprintf("%s↓ Scroll down for more files%s\n", colorGrey, colorReset))
	}

	// Single write operation
	output.Write(buf.Bytes())
	output.Flush()
}

func getRealIndex(entries []FileEntry, name string) int {
	for i, entry := range entries {
		if entry.name == name {
			return i
		}
	}
	return 0
}

func joinSelectedFiles(listsDir string, selectedEntries []FileEntry) error {
	ultimatePath := filepath.Join(listsDir, "list-ultimate.txt")
	ultimateFile, err := os.Create(ultimatePath)
	if err != nil {
		return err
	}
	defer ultimateFile.Close()

	for _, entry := range selectedEntries {
		if !entry.isControl {
			filePath := filepath.Join(listsDir, entry.name)
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			fmt.Fprintln(ultimateFile, strings.TrimSpace(string(content)))
		}
	}

	return nil
}

func main() {
	var buf bytes.Buffer
	buf.Grow(bufferSize)

	// Create output buffer for direct writes
	output := bufio.NewWriter(os.Stdout)
	defer output.Flush()

	// Initialize terminal
	buf.WriteString("\033[H\033[J")
	output.Write(buf.Bytes())
	output.Flush()

	fmt.Print(enterAltScreen + hideCursor)
	defer fmt.Print(showCursor + exitAltScreen)

	setupTerminalCleanup()

	listsDir := "lists"
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		fmt.Printf("Error creating lists directory: %v\n", err)
		return
	}

	// Load saved selections
	var selectedFiles []string
	if content, err := os.ReadFile(filepath.Join(listsDir, "selected.txt")); err == nil {
		selectedFiles = strings.Split(strings.TrimSpace(string(content)), "\n")
	}

	// Create file list
	entries := []FileEntry{
		{name: "SAVE LIST", isControl: true},
		{name: "CANCEL", isControl: true},
	}

	files, err := os.ReadDir(listsDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, "list-") && strings.HasSuffix(name, ".txt") && name != "list-ultimate.txt" {
			entries = append(entries, FileEntry{
				name:     name,
				selected: contains(selectedFiles, name),
			})
		}
	}

	if err := keyboard.Open(); err != nil {
		fmt.Printf("Error initializing keyboard: %v\n", err)
		return
	}
	defer keyboard.Close()

	var currentIndex, scrollOffset int

	for {
		start := time.Now()

		// Draw screen using buffered output
		drawScreen(&buf, entries, currentIndex, scrollOffset, output)

		// Precise frame timing
		elapsed := time.Since(start)
		if elapsed < frameTime {
			time.Sleep(frameTime - elapsed)
		}

		_, key, err := keyboard.GetKey()
		if err != nil {
			fmt.Printf("Error reading keyboard: %v\n", err)
			return
		}

		switch key {
		case keyboard.KeyArrowUp:
			if currentIndex > 0 {
				currentIndex--
				if currentIndex >= 2 && currentIndex-2 < scrollOffset {
					scrollOffset = currentIndex - 2
				}
			}
		case keyboard.KeyArrowDown:
			if currentIndex < len(entries)-1 {
				currentIndex++
				if currentIndex >= 2 && currentIndex-2 >= scrollOffset+visibleItems {
					scrollOffset = currentIndex - visibleItems + 1 - 2
				}
			}
		case keyboard.KeySpace, keyboard.KeyEnter:
			if entries[currentIndex].isControl {
				switch entries[currentIndex].name {
				case "SAVE LIST":
					// Save selected files
					selectedFile, err := os.Create(filepath.Join(listsDir, "selected.txt"))
					if err == nil {
						for _, entry := range entries {
							if entry.selected {
								fmt.Fprintln(selectedFile, entry.name)
							}
						}
						selectedFile.Close()

						// Merge selected files
						var selectedEntries []FileEntry
						for _, entry := range entries {
							if entry.selected {
								selectedEntries = append(selectedEntries, entry)
							}
						}

						if err := joinSelectedFiles(listsDir, selectedEntries); err != nil {
							fmt.Printf("\n%sError occurred while merging files: %v. Exiting in 5 seconds...%s\n",
								colorRed, err, colorReset)
						} else {
							fmt.Printf("\n%sSuccessful! List saved and files merged. Exiting in 5 seconds...%s\n",
								colorGreen, colorReset)
						}
						time.Sleep(5 * time.Second)
						return
					}
				case "CANCEL":
					return
				}
			} else {
				entries[currentIndex].selected = !entries[currentIndex].selected
			}
		case keyboard.KeyEsc:
			return
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
