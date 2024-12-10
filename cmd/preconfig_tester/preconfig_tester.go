package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[38;2;126;176;0m"   // RGB(126,176,0)
	colorRed     = "\033[38;2;214;77;91m"   // RGB(214,77,91)
	colorMagenta = "\033[38;2;196;124;186m" // RGB(196,124,186)

	// Terminal control
	enterAltScreen = "\033[?1049h"
	exitAltScreen  = "\033[?1049l"
	hideCursor     = "\033[?25l"
	showCursor     = "\033[?25h"
	clearScreen    = "\033[2J\033[H"
)

var domainList = []struct {
	number string
	domain string
}{
	{"1", "discord.com"},
	{"2", "youtube.com"},
	{"3", "spotify.com"},
	{"4", "speedtest.net"},
	{"5", "steampowered.com"},
	{"6", "custom"},
	{"0", "exit"},
}

type Config struct {
	batchDir          string
	targetDomain      string
	processName       string
	processWaitTime   time.Duration
	connectionTimeout time.Duration
}

type DPITestResult int

const (
	NoDPI DPITestResult = iota
	HasDPI
	NoConnection
)

func (r DPITestResult) String() string {
	switch r {
	case NoDPI:
		return "No DPI detected"
	case HasDPI:
		return "DPI blocks detected"
	case NoConnection:
		return "No connection"
	default:
		return "Unknown"
	}
}

func (c *Config) getBatchFiles() ([]string, error) {
	var batFiles []string
	files, err := os.ReadDir(c.batchDir)
	if err != nil {
		return nil, fmt.Errorf("error reading batch directory: %v", err)
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".bat") {
			batFiles = append(batFiles, filepath.Join(c.batchDir, f.Name()))
		}
	}
	return batFiles, nil
}

func isElevated() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

func requestElevation() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	cmd := exec.Command("powershell", "Start-Process", executable, "-Verb", "RunAs", "-ArgumentList", "--elevated")
	return cmd.Run()
}

func ensureProcessTerminated(processName string) {
	cmd := exec.Command("taskkill", "/F", "/IM", processName)
	cmd.Run() // Игнорируем ошибки, так как процесс может не существовать
}

func waitForProcess(processName string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
		output, _ := cmd.Output()
		if strings.Contains(string(output), processName) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func testConnection(domain string, timeout time.Duration) bool {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
		$webRequest = [System.Net.WebRequest]::Create("https://%s")
		$webRequest.Timeout = %d
		try {
			$response = $webRequest.GetResponse()
			$response.Close()
			return $true
		} catch {
			return $false
		}
	`, domain, int(timeout.Milliseconds())))

	err := cmd.Run()
	return err == nil
}

func getDomainChoice() (string, error) {
	fmt.Println("\nSelect domain for checking:")
	for _, item := range domainList {
		if item.domain == "exit" {
			fmt.Printf("%s. Exit\n", item.number)
		} else if item.domain == "custom" {
			fmt.Printf("%s. Enter your own domain\n", item.number)
		} else {
			fmt.Printf("%s. %s\n", item.number, item.domain)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nEnter number of variant: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("error reading input: %v", err)
		}
		choice = strings.TrimSpace(choice)

		for _, item := range domainList {
			if item.number == choice {
				if item.domain == "exit" {
					fmt.Print("Exiting..")
					fmt.Print(showCursor + exitAltScreen)
					os.Exit(0)
				}
				if item.domain == "custom" {
					fmt.Print("Enter domain (for example, example.com): ")

					domain, err := reader.ReadString('\n')
					if err != nil {
						return "", err
					}
					domain = strings.TrimSpace(domain)
					if isValidDomain(domain) {
						return formatDomainWithPort(domain), nil
					}
					fmt.Println("Invalid domain format. Use format domain.com")
					continue
				}
				return formatDomainWithPort(item.domain), nil
			}
		}
		fmt.Printf("Invalid selection. Please select number from 0 to %d\n", len(domainList)-1)
	}
}

const DEFAULT_PORT = 443

func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 255 {
		return false
	}

	if strings.Contains(domain, ":") {
		return false
	}

	// Check if domain contains only allowed characters
	for _, c := range domain {
		if !isAsciiAlphanumeric(c) && c != '.' && c != '-' {
			return false
		}
	}

	// Check that domain doesn't start or end with hyphen
	return !strings.HasPrefix(domain, "-") && !strings.HasSuffix(domain, "-")
}

func isAsciiAlphanumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func formatDomainWithPort(domain string) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(domain), DEFAULT_PORT)
}

func checkDPIFingerprint(domain string) (DPITestResult, error) {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
		try {
			$webRequest = [System.Net.WebRequest]::Create("https://%s")
			$webRequest.Timeout = 5000
			$response = $webRequest.GetResponse()
			$response.Close()
			return 0  # NoDPI
		} catch [System.Net.WebException] {
			if ($_.Exception.Message -like "*actively refused*") {
				return 1  # HasDPI
			}
			return 2  # NoConnection
		}
	`, domain))

	output, err := cmd.Output()
	if err != nil {
		return NoConnection, err
	}

	result, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return NoConnection, err
	}

	return DPITestResult(result), nil
}

func runBypassCheck(config Config) error {
	domainWithoutPort := strings.Split(config.targetDomain, ":")[0]

	fmt.Printf("\nStarting testing domain: %s\n", config.targetDomain)
	fmt.Println("------------------------------------------------")

	fmt.Println("Checking DPI blocks...")
	result, err := checkDPIFingerprint(domainWithoutPort)
	if err != nil {
		fmt.Printf("Error occurred while checking: %v\n", err)
	} else {
		fmt.Printf("Checking result: %s\n", result)

		if result == NoDPI {
			fmt.Println("Using DPI spoofer not required.")
			return nil
		}

		if result == NoConnection {
			fmt.Println("Check internet connection and if domain is correct.")
			return nil
		}
	}

	fmt.Println("------------------------------------------------")
	fmt.Println("Testing pre-configs...")

	batFiles, err := config.getBatchFiles()
	if err != nil {
		return err
	}

	success := false
	for _, batFile := range batFiles {
		fmt.Printf("\n%sRunning pre-config: %s%s\n", colorMagenta, batFile, colorReset)

		// Ensure no previous process is running
		ensureProcessTerminated(config.processName)

		cmd := exec.Command("cmd", "/c", batFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			fmt.Printf("%sFailed to run pre-config %s: %v%s\n", colorRed, batFile, err, colorReset)
			continue
		}

		if !waitForProcess(config.processName, config.processWaitTime) {
			fmt.Printf("%s%s not started for pre-config %s%s\n", colorRed, config.processName, batFile, colorReset)
			cmd.Process.Kill()
			continue
		}

		if testConnection(config.targetDomain, config.connectionTimeout) {
			filename := filepath.Base(batFile)
			fmt.Printf("\n%s!!!!!!!!!!!!!\n[SUCCESS] It seems, this pre-config is suitable for you - %s\n!!!!!!!!!!!!!\n%s\n",
				colorGreen, filename, colorReset)
			cmd.Process.Kill()
			success = true
			break
		} else {
			fmt.Printf("%s[FAIL] Failed to establish connection using pre-config: %s%s\n",
				colorRed, batFile, colorReset)
			cmd.Process.Kill()
		}
	}

	// Final cleanup
	ensureProcessTerminated(config.processName)
	time.Sleep(500 * time.Millisecond)
	ensureProcessTerminated(config.processName)

	if !success {
		fmt.Println("\n------------------------------------------------")
		fmt.Println("Unfortunately, not found pre-config we can establish connection with :(")
		fmt.Println("Try to run BLOCKCHECK, to find necessary parameters for BAT file.")
	}

	return nil
}

func main() {
	// Terminal initialization
	fmt.Print(enterAltScreen + hideCursor)
	// Clear screen before switching
	fmt.Print(clearScreen)
	// Restore normal terminal state on exit
	defer fmt.Print(showCursor + exitAltScreen)

	// Check for administrator privileges
	if !isElevated() {
		fmt.Print("Administrative privileges required for correct work of program.\n" +
			"Please, confirm prompt for administrative privileges.")
		if err := requestElevation(); err != nil {
			fmt.Printf("\nError occurred while requesting administrative privileges: %v", err)
			time.Sleep(3 * time.Second)
			return
		}
		fmt.Print(showCursor + exitAltScreen)
		os.Exit(0)
	}

	targetDomain, err := getDomainChoice()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	config := Config{
		batchDir:          "pre-configs",
		targetDomain:      targetDomain,
		processName:       "winws.exe",
		processWaitTime:   10 * time.Second,
		connectionTimeout: 5 * time.Second,
	}

	if err := runBypassCheck(config); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\nPress Enter to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
