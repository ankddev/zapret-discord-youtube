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

// buildConfig is the configuration for the build process
type buildConfig struct {
	currentDir    string
	buildDir      string
	binDir        string
	listsDir      string
	preConfigsDir string
	zipPath       string
}

func main() {
	if !isWindows() {
		os.Exit(1)
	}

	config, err := initBuildConfig()
	if err != nil {
		fmt.Printf("Error initializing build config: %v\n", err)
		os.Exit(1)
	}

	if err := runBuildSteps(config); err != nil {
		fmt.Printf("Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nRelease build ready! Check '%s'\n", config.zipPath)
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}

// initBuildConfig initializes the build configuration
func initBuildConfig() (*buildConfig, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %v", err)
	}

	buildDir := filepath.Join(currentDir, "build")
	return &buildConfig{
		currentDir:    currentDir,
		buildDir:      buildDir,
		binDir:        filepath.Join(currentDir, "bin"),
		listsDir:      filepath.Join(currentDir, "lists"),
		preConfigsDir: filepath.Join(currentDir, "pre-configs"),
		zipPath:       filepath.Join(buildDir, "zapret-discord-youtube-ankddev.zip"),
	}, nil
}

// runBuildSteps runs the build steps
func runBuildSteps(config *buildConfig) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.HideCursor = true

	// Step 1: Build
	s.Suffix = " [1/4] Building..."
	s.FinalMSG = "[1/4] Build successful\n"
	s.Start()
	if err := buildExecutables(); err != nil {
		return fmt.Errorf("build failed: %v", err)
	}
	s.Stop()

	// Step 2: Check paths and create zip
	if err := validatePaths(config); err != nil {
		return err
	}

	zipFile, zipWriter, err := createZipFile(config.zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	defer zipWriter.Close()

	// Step 3: Add directories
	s.Suffix = " [2/4] Adding directories..."
	s.FinalMSG = "[2/4] Directories added\n"
	s.Start()
	if err := addDirectories(zipWriter, config); err != nil {
		return err
	}
	s.Stop()

	// Step 4: Add files
	s.Suffix = " [3/4] Adding files..."
	s.FinalMSG = "[3/4] Files added\n"
	s.Start()
	if err := addFiles(zipWriter, config); err != nil {
		return err
	}
	s.Stop()

	fmt.Println("[4/4] Release archive created successfully!")
	return nil
}

// buildExecutables builds the executables
func buildExecutables() error {
	_ = os.Mkdir("build", os.ModePerm)
	ldflags := fmt.Sprintf("-X main.version=%s %s", version(), os.Getenv("GO_LDFLAGS"))
	return run("go", "build", "-ldflags", ldflags, "-o", "build", "./cmd/...")
}

// validatePaths validates the paths
func validatePaths(config *buildConfig) error {
	paths := []string{config.buildDir, config.listsDir, config.preConfigsDir}
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("required directory not found: %s", path)
		}
	}
	return nil
}

// createZipFile creates the zip file
func createZipFile(zipPath string) (*os.File, *zip.Writer, error) {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating zip file: %v", err)
	}
	return zipFile, zip.NewWriter(zipFile), nil
}

// addDirectories adds the directories to the zip file
func addDirectories(zipWriter *zip.Writer, config *buildConfig) error {
	dirsToAdd := map[string]string{
		"lists":       config.listsDir,
		"pre-configs": config.preConfigsDir,
		"bin":         config.binDir,
	}

	for zipPath, fsPath := range dirsToAdd {
		if err := addDirToZip(zipWriter, zipPath, fsPath); err != nil {
			return fmt.Errorf("error adding directory %s: %v", fsPath, err)
		}
	}
	return nil
}

// addFiles adds the files to the zip file
func addFiles(zipWriter *zip.Writer, config *buildConfig) error {
	filesToAdd := map[string]string{
		"blockcheck.cmd":                      filepath.Join(config.currentDir, "blockcheck.cmd"),
		"Add to autorun.exe":                  filepath.Join(config.buildDir, "add_to_autorun.exe"),
		"Automatically search pre-config.exe": filepath.Join(config.buildDir, "preconfig_tester.exe"),
		"Run pre-config.exe":                  filepath.Join(config.buildDir, "run_preconfig.exe"),
		"Set domain list.exe":                 filepath.Join(config.buildDir, "select_domains.exe"),
		"Check for updates.exe":               filepath.Join(config.buildDir, "check_for_updates.exe"),
	}

	for zipPath, fsPath := range filesToAdd {
		if err := addFileToZip(zipWriter, zipPath, fsPath); err != nil {
			return fmt.Errorf("error adding file %s: %v", fsPath, err)
		}
	}
	return nil
}

// addDirToZip adds the directory to the zip file
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

// addFileToZip adds the file to the zip file
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

// cmdOutput runs a command and returns its output
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

// run runs a command
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
