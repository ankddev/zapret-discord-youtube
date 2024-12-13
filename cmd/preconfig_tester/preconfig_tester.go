package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

	fps        = 240
	frameTime  = time.Second / time.Duration(fps)
	bufferSize = 4096
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
	{"7", "custom_multiple"},
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

	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
		$proc = Start-Process -FilePath "%s" -Verb RunAs -PassThru -WindowStyle Normal
		if ($proc.ExitCode -ne 0) {
			exit $proc.ExitCode
		}
	`, executable))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func ensureProcessTerminated(processName string) {
	cmd := exec.Command("taskkill", "/F", "/IM", processName)
	cmd.Run() // Ignore errors as the process may not exist
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

func getDomainChoice() ([]string, error) {
	fmt.Println("\nSelect domain for checking:")
	for _, item := range domainList {
		switch item.domain {
		case "exit":
			fmt.Printf("%s. Exit\n", item.number)
		case "custom":
			fmt.Printf("%s. Enter your own domain\n", item.number)
		case "custom_multiple":
			fmt.Printf("%s. Enter multiple domains (space-separated)\n", item.number)
		default:
			fmt.Printf("%s. %s\n", item.number, item.domain)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nEnter number of variant: ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading input: %v", err)
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
						return nil, err
					}
					domain = strings.TrimSpace(domain)
					if isValidDomain(domain) {
						return []string{formatDomainWithPort(domain)}, nil
					}
					fmt.Println("Invalid domain format. Use format domain.com")
					continue
				}
				if item.domain == "custom_multiple" {
					fmt.Print("Enter domains separated by spaces: ")
					domains, err := reader.ReadString('\n')
					if err != nil {
						return nil, err
					}

					domainList := strings.Fields(domains)
					var formattedDomains []string

					for _, domain := range domainList {
						if !isValidDomain(domain) {
							fmt.Printf("Invalid domain format for '%s'. Use format domain.com\n", domain)
							continue
						}
						formattedDomains = append(formattedDomains, formatDomainWithPort(domain))
					}

					if len(formattedDomains) > 0 {
						return formattedDomains, nil
					}
					continue
				}
				return []string{formatDomainWithPort(item.domain)}, nil
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
		# Save console settings
		$originalForeground = $host.UI.RawUI.ForegroundColor
		$originalBackground = $host.UI.RawUI.BackgroundColor
		
		# Force TLS 1.2
		[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

		try {
			$url = "https://%s"
			Write-Information "Trying to connect to $url"
			
			$webRequest = [System.Net.WebRequest]::Create($url)
			$webRequest.Timeout = 5000
			$webRequest.AllowAutoRedirect = $false
			
			try {
				$response = $webRequest.GetResponse()
				$response.Close()
				Write-Output "0"  # NoDPI
			} catch [System.Net.WebException] {
				$exception = $_.Exception
				Write-Information "Exception details: $($exception.Message)"
				Write-Information "Status: $($exception.Status)"
				
				if ($exception.Message -like "*actively refused*" -or 
					$exception.Message -like "*connection was forcibly closed*" -or
					$exception.Status -eq [System.Net.WebExceptionStatus]::SecureChannelFailure -or
					$exception.Status -eq [System.Net.WebExceptionStatus]::TrustFailure -or
					$exception.Status -eq [System.Net.WebExceptionStatus]::ProtocolError) {
					Write-Output "1"  # HasDPI
				} elseif ($exception.Status -eq [System.Net.WebExceptionStatus]::NameResolutionFailure) {
					Write-Output "2"  # DNS resolution failed
					Write-Information "DNS resolution failed"
				} elseif ($exception.Status -eq [System.Net.WebExceptionStatus]::Timeout) {
					Write-Output "1"  # Treat timeout as potential DPI
					Write-Information "Connection timed out - possible DPI"
				} else {
					Write-Output "2"  # NoConnection
					Write-Information "Unknown connection error"
				}
			}
		} catch {
			Write-Output "2"  # NoConnection
			Write-Information "Unexpected error: $_"
		} finally {
			# Restore console settings
			$host.UI.RawUI.ForegroundColor = $originalForeground
			$host.UI.RawUI.BackgroundColor = $originalBackground
		}
		exit 0
	`, domain))

	cmd.Stderr = os.Stderr // Show diagnostic output
	output, err := cmd.Output()
	if err != nil {
		if len(output) == 0 {
			return NoConnection, fmt.Errorf("no output from DPI check: %v", err)
		}
	}

	// Берем только последнюю строку вывода, которая содержит результат
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result, err := strconv.Atoi(lines[len(lines)-1])
	if err != nil {
		return NoConnection, fmt.Errorf("invalid output from DPI check: %v", err)
	}

	return DPITestResult(result), nil
}

func runBypassCheck(config Config) error {
	domains := strings.Split(config.targetDomain, " ")

	fmt.Printf("\nStarting testing domains: %s\n", config.targetDomain)
	fmt.Println("------------------------------------------------")

	// Check DPI for each domain
	for _, domain := range domains {
		// Remove port before DPI check but keep it for display
		domainForCheck := strings.Split(domain, ":")[0]
		fmt.Printf("\nChecking DPI blocks for %s...\n", domainForCheck)
		result, err := checkDPIFingerprint(domainForCheck)
		if err != nil {
			fmt.Printf("Checking result for %s: %s (with error: %v)\n", domainForCheck, result, err)
		} else {
			fmt.Printf("Checking result for %s: %s\n", domainForCheck, result)
		}

		if result == NoDPI {
			fmt.Printf("Using DPI spoofer not required for %s.\n", domainForCheck)
			continue
		}

		if result == NoConnection {
			fmt.Printf("Check internet connection and if domain %s is correct.\n", domainForCheck)
			continue
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

		ensureProcessTerminated(config.processName)

		cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
			# Save console settings
			$originalForeground = $host.UI.RawUI.ForegroundColor
			$originalBackground = $host.UI.RawUI.BackgroundColor
			$originalBufferSize = $host.UI.RawUI.BufferSize
			$originalWindowSize = $host.UI.RawUI.WindowSize

			# Save font settings
			$key = 'HKCU:\Console'
			$originalFontSize = Get-ItemProperty -Path $key -Name 'FontSize' -ErrorAction SilentlyContinue
			$originalFaceName = Get-ItemProperty -Path $key -Name 'FaceName' -ErrorAction SilentlyContinue
			$originalFontFamily = Get-ItemProperty -Path $key -Name 'FontFamily' -ErrorAction SilentlyContinue
			
			try {
				# Execute BAT file
				cmd /c "%s"
			} finally {
				# Restore console settings
				$host.UI.RawUI.ForegroundColor = $originalForeground
				$host.UI.RawUI.BackgroundColor = $originalBackground
				$host.UI.RawUI.BufferSize = $originalBufferSize
				$host.UI.RawUI.WindowSize = $originalWindowSize

				# Restore font settings
				if ($originalFontSize) {
					Set-ItemProperty -Path $key -Name 'FontSize' -Value $originalFontSize.FontSize
				}
				if ($originalFaceName) {
					Set-ItemProperty -Path $key -Name 'FaceName' -Value $originalFaceName.FaceName
				}
				if ($originalFontFamily) {
					Set-ItemProperty -Path $key -Name 'FontFamily' -Value $originalFontFamily.FontFamily
				}
			}
		`, batFile))

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

		// Check all domains
		allDomainsWork := true
		for _, domain := range domains {
			if !testConnection(domain, config.connectionTimeout) {
				fmt.Printf("%s[FAIL] Failed to establish connection to %s using pre-config: %s%s\n",
					colorRed, domain, batFile, colorReset)
				allDomainsWork = false
				break
			}
		}

		if allDomainsWork {
			filename := filepath.Base(batFile)
			fmt.Printf("\n%s!!!!!!!!!!!!!\n[SUCCESS] It seems, this pre-config is suitable for all specified domains - %s\n!!!!!!!!!!!!!\n%s\n",
				colorGreen, filename, colorReset)
			cmd.Process.Kill()
			success = true
			break
		}

		cmd.Process.Kill()
	}

	ensureProcessTerminated(config.processName)
	time.Sleep(500 * time.Millisecond)
	ensureProcessTerminated(config.processName)

	if !success {
		fmt.Println("\n------------------------------------------------")
		fmt.Println("Unfortunately, not found pre-config we can establish connection with for all specified domains :(")
		fmt.Println("Try to run BLOCKCHECK, to find necessary parameters for BAT file.")
	}

	return nil
}

func main() {
	// Add signal handling at the start of main
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print(showCursor + exitAltScreen)
		os.Exit(1)
	}()

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
		// Wait a bit to ensure new process starts
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}

	targetDomains, err := getDomainChoice()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	config := Config{
		batchDir:          "pre-configs",
		targetDomain:      strings.Join(targetDomains, " "),
		processName:       "winws.exe",
		processWaitTime:   10 * time.Second,
		connectionTimeout: 5 * time.Second,
	}

	// Use buffered output for all writes
	buf.Reset()
	buf.WriteString(fmt.Sprintf("\nStarting testing domains: %s\n", config.targetDomain))
	output.Write(buf.Bytes())
	output.Flush()

	if err := runBypassCheck(config); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\nPress Enter to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
