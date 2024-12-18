package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cli/safeexec"
)

func main() {
	if !isWindows() {
		os.Exit(1)
	}

	s.Suffix = " [1/4] Building..."
	s.FinalMSG = "[3/4] Building...\n"
	s.HideCursor = true
	s.Start()
	ldflags := os.Getenv("GO_LDFLAGS")
	ldflags = fmt.Sprintf("-X main.version=%s %s", version(), ldflags)
	_ = os.Mkdir("build", os.ModePerm)
	err := run("go", "build", "-ldflags", ldflags, "-o", "build", "./cmd/...")
	if err != nil {
		fmt.Println("Build failed:", err)
		os.Exit(1)
	}
        s.Stop()

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Define paths
	buildDir := filepath.Join(currentDir, "build")
	binDir := filepath.Join(currentDir, "bin")
	listsDir := filepath.Join(currentDir, "lists")
	preConfigsDir := filepath.Join(currentDir, "pre-configs")

	// Check required paths exist
	requiredPaths := []string{
		buildDir,
		listsDir,
		preConfigsDir,
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("Required directory not found: %s\n", path)
			os.Exit(1)
		}
	}

	// Create zip file
	zipPath := filepath.Join(buildDir, "zapret-discord-youtube-ankddev.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		fmt.Printf("Error creating zip file: %v\n", err)
		os.Exit(1)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add directories to zip
	dirsToAdd := map[string]string{
		"lists":       listsDir,
		"pre-configs": preConfigsDir,
		"bin":         binDir,
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " [2/4] Adding directories..."
	s.FinalMSG = "[2/4] Adding directories...\n"
	s.HideCursor = true
	s.Start()
	for zipPath, fsPath := range dirsToAdd {
		err = addDirToZip(zipWriter, zipPath, fsPath)
		if err != nil {
			fmt.Printf("Error adding directory %s to zip: %v\n", fsPath, err)
			os.Exit(1)
		}
	}
	s.Stop()

	// Add individual files
	s.Suffix = " [3/4] Adding files..."
	s.FinalMSG = "[3/4] Adding files...\n"
	s.HideCursor = true
	s.Start()
	filesToAdd := map[string]string{
		"blockcheck.cmd":                      filepath.Join(currentDir, "blockcheck.cmd"),
		"Add to autorun.exe":                  filepath.Join(buildDir, "add_to_autorun.exe"),
		"Automatically search pre-config.exe": filepath.Join(buildDir, "preconfig_tester.exe"),
		"Run pre-config.exe":                  filepath.Join(buildDir, "run_preconfig.exe"),
		"Set domain list.exe":                 filepath.Join(buildDir, "select_domains.exe"),
		"Check for updates.exe":               filepath.Join(buildDir, "check_for_updates.exe"),
	}

	for zipPath, fsPath := range filesToAdd {
		err = addFileToZip(zipWriter, zipPath, fsPath)
		if err != nil {
			fmt.Printf("Error adding file %s to zip: %v\n", fsPath, err)
			os.Exit(1)
		}
	}

	s.Stop()

	fmt.Println("[4/4] Release archive created successfully!")
	fmt.Printf("\nRelease build ready! Check '%s'\n", zipPath)
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}

func addDirToZip(zipWriter *zip.Writer, zipPath string, fsPath string) error {
	return filepath.Walk(fsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get zip path for file
		relativePath, err := filepath.Rel(fsPath, path)
		if err != nil {
			return err
		}
		zipEntryPath := filepath.Join(zipPath, relativePath)

		if info.IsDir() {
			return nil
		}

		return addFileToZip(zipWriter, zipEntryPath, path)
	})
}

func addFileToZip(zipWriter *zip.Writer, zipPath string, fsPath string) error {
	file, err := os.Open(fsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// version returns version from environment variable or git describe
func version() string {
	if versionEnv := os.Getenv("VERSION"); versionEnv != "" {
		return versionEnv
	}
	if content, err := os.ReadFile(".service/version.txt"); err == nil {
		return strings.TrimSpace("v" + string(content))
	}
	if desc, err := cmdOutput("git", "describe", "--tags"); err == nil {
		return desc
	}
	rev, _ := cmdOutput("git", "rev-parse", "--short", "HEAD")
	return rev
}

func cmdOutput(args ...string) (string, error) {
	exe, err := safeexec.LookPath(args[0])
	if err != nil {
		return "", err
	}
	cmd := exec.Command(exe, args[1:]...)
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	return strings.TrimSuffix(string(out), "\n"), err
}

func isWindows() bool {
	if os.Getenv("GOOS") == "windows" {
		return true
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return false
}

func run(args ...string) error {
	exe, err := safeexec.LookPath(args[0])
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellInspect(args []string) string {
	fmtArgs := make([]string, len(args))
	for i, arg := range args {
		if strings.ContainsAny(arg, " \t'\"") {
			fmtArgs[i] = fmt.Sprintf("%q", arg)
		} else {
			fmtArgs[i] = arg
		}
	}
	return strings.Join(fmtArgs, " ")
}
