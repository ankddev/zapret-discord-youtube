package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// Terminal control sequences
	enterAltScreen = "\033[?1049h"
	exitAltScreen  = "\033[?1049l"
	hideCursor     = "\033[?25l"
	showCursor     = "\033[?25h"
	clearScreen    = "\033[2J\033[H"

	// Colors for terminal output
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"

	// GitHub API endpoints
	githubAPI    = "https://api.github.com/repos/ankddev/zapret-discord-youtube"
	versionFile  = "https://raw.githubusercontent.com/ankddev/zapret-discord-youtube/main/.service/version.txt"
	releaseAsset = "zapret-discord-youtube-ankddev.zip"

	fps        = 240
	frameTime  = time.Second / time.Duration(fps)
	bufferSize = 4096
)

// Version is set during build
var version string

// Release represents GitHub release information
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func setupTerminalCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print(showCursor + exitAltScreen)
		os.Exit(1)
	}()
}

// Get current version from version.txt
func getCurrentVersion() (string, error) {
	resp, err := http.Get(versionFile)
	if err != nil {
		return "", fmt.Errorf("failed to get version file: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version file: %v", err)
	}

	return strings.TrimSpace(string(body)), nil
}

// Get latest release information from GitHub
func getLatestRelease() (*Release, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", githubAPI+"/releases/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add User-Agent header to avoid GitHub API limitations
	req.Header.Set("User-Agent", "zapret-discord-youtube-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %v", err)
	}

	return &release, nil
}

// Download the release asset with progress indication
func downloadRelease(url, destPath string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "zapret-discord-youtube-updater")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	fmt.Print("\nDownloading... ")
	_, err = io.Copy(out, resp.Body)
	fmt.Println("Done!")

	return err
}

func main() {
	// Initialize terminal
	var buf bytes.Buffer
	buf.Grow(bufferSize)

	// Create output buffer for direct writes
	output := bufio.NewWriter(os.Stdout)
	defer output.Flush()

	// Use single write operations
	buf.WriteString("\033[H\033[J")
	output.Write(buf.Bytes())
	output.Flush()

	fmt.Print(enterAltScreen + hideCursor)
	defer fmt.Print(showCursor + exitAltScreen)

	setupTerminalCleanup()

	// Remove 'v' prefix from current version for comparison
	currentVersion := strings.TrimPrefix(version, "v")

	// Get version from GitHub
	remoteVersion, err := getCurrentVersion()
	if err != nil {
		fmt.Printf("%sError checking for updates: %v%s\n", colorRed, err, colorReset)
		waitForEnter()
		return
	}

	remoteVersion = strings.TrimPrefix(remoteVersion, "v")

	// Compare versions using semver
	if semver.Compare("v"+currentVersion, "v"+remoteVersion) >= 0 {
		fmt.Printf("%sYou have the latest version (%s)%s\n", colorGreen, version, colorReset)
		waitForEnter()
		return
	}

	// Ask user about update
	fmt.Printf("%sNew version available: %s (current: %s)%s\n", colorCyan, remoteVersion, currentVersion, colorReset)
	fmt.Print("Do you want to download the update? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("Update cancelled.")
		waitForEnter()
		return
	}

	// Get latest release info
	release, err := getLatestRelease()
	if err != nil {
		fmt.Printf("%sError getting release information: %v%s\n", colorRed, err, colorReset)
		waitForEnter()
		return
	}

	// Find download URL
	var downloadURL string
	for _, asset := range release.Assets {
		expectedAsset := fmt.Sprintf("zapret-discord-youtube-ankddev-%s.zip", release.TagName)
		if asset.Name == expectedAsset {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		fmt.Printf("%sError: Release asset not found%s\n", colorRed, colorReset)
		waitForEnter()
		return
	}

	// Create downloads directory in parent folder
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("%sError getting current directory: %v%s\n", colorRed, err, colorReset)
		waitForEnter()
		return
	}

	parentDir := filepath.Dir(currentDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		fmt.Printf("%sError creating download directory: %v%s\n", colorRed, err, colorReset)
		waitForEnter()
		return
	}

	// Download the release with version in filename
	destPath := filepath.Join(parentDir, fmt.Sprintf("zapret-discord-youtube-ankddev-%s.zip", release.TagName))
	absPath, err := filepath.Abs(destPath)
	if err != nil {
		fmt.Printf("%sError getting absolute path: %v%s\n", colorRed, err, colorReset)
		waitForEnter()
		return
	}

	fmt.Printf("Downloading update to: %s\n", absPath)

	if err := downloadRelease(downloadURL, destPath); err != nil {
		fmt.Printf("%sError downloading update: %v%s\n", colorRed, err, colorReset)
		waitForEnter()
		return
	}

	fmt.Printf("%sUpdate downloaded successfully to: %s%s\n", colorGreen, absPath, colorReset)
	waitForEnter()
}

func waitForEnter() {
	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
