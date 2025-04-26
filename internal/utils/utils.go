package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// FileExists checks if a file exists at the given path
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// CreateDir creates a directory if it doesn't exist
func CreateDir(path string) error {
	if DirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dst)
	if err := CreateDir(dstDir); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Preserve file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// CopyDir recursively copies a directory
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			if err = CopyDir(sourcePath, destPath); err != nil {
				return err
			}
		} else {
			if err = CopyFile(sourcePath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveDir removes a directory and all its contents
func RemoveDir(path string) error {
	return os.RemoveAll(path)
}

// ExecuteCommand executes a command with the given arguments
func ExecuteCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// ExecuteCommandWithOutput executes a command and returns its output
func ExecuteCommandWithOutput(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// IsGodspeedProject checks if the current directory is a godspeed project
func IsGodspeedProject() bool {
	// Check for .godspeed file
	if !FileExists(".godspeed") {
		color.Red("The current directory is not a Godspeed Framework project.")
		color.Yellow("godspeed commands work inside godspeed project directory.")
		return false
	}

	// Check for package.json
	if !FileExists("package.json") {
		color.Red("The current directory is not a Godspeed project.")
		color.Yellow("godspeed commands only work inside godspeed project directory.")
		return false
	}

	return true
}

// UserHomeDir returns the user's home directory
func UserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Could not determine user home directory:", err)
		os.Exit(1)
	}
	return home
}

// GetGodspeedDir returns the .godspeed directory in the user's home
func GetGodspeedDir() string {
	return filepath.Join(UserHomeDir(), ".godspeed")
}

// NewSpinner creates a new spinner with godspeed style
func NewSpinner(text string) *spinner.Spinner {
	s := spinner.New([]string{"🌍 ", "🌎 ", "🌏 ", "🌐 ", "🌑 ", "🌒 ", "🌓 ", "🌔 "}, 180*time.Millisecond)
	s.Prefix = text
	return s
}

// ServicesJson represents the structure of services.json
type ServicesJson struct {
	Services []Service `json:"services"`
}

// Service represents a godspeed service
type Service struct {
	ServiceID   string `json:"serviceId"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	LastUpdated string `json:"last_updated"`
	Initialized bool   `json:"initialized"`
}

// UpdateServicesJson updates the services.json file to add or remove the current project
func UpdateServicesJson(add bool) {
	servicesFile := filepath.Join(GetGodspeedDir(), "services.json")

	// If services.json doesn't exist, return early if removing
	if !FileExists(servicesFile) && !add {
		return
	}

	var servicesData ServicesJson
	if FileExists(servicesFile) {
		data, err := os.ReadFile(servicesFile)
		if err != nil {
			color.Red("Error reading services.json: %v", err)
			return
		}

		if err := json.Unmarshal(data, &servicesData); err != nil {
			color.Red("Error parsing services.json: %v", err)
			return
		}
	}

	currentDir, err := os.Getwd()
	if err != nil {
		color.Red("Error getting current directory: %v", err)
		return
	}

	currentProject := Service{
		ServiceID:   filepath.Base(currentDir),
		Name:        filepath.Base(currentDir),
		Path:        currentDir,
		Status:      "active",
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Initialized: true,
	}

	if add {
		// Check if the project already exists
		exists := false
		for _, service := range servicesData.Services {
			if service.Path == currentDir {
				exists = true
				break
			}
		}

		if !exists {
			servicesData.Services = append(servicesData.Services, currentProject)
		}
	} else {
		// Remove the project if it exists
		var newServices []Service
		for _, service := range servicesData.Services {
			if service.Path != currentDir {
				newServices = append(newServices, service)
			}
		}
		servicesData.Services = newServices
	}

	// Write the updated services.json
	updatedData, err := json.MarshalIndent(servicesData, "", "  ")
	if err != nil {
		color.Red("Error creating JSON: %v", err)
		return
	}

	// Create .godspeed directory if needed
	if err := CreateDir(GetGodspeedDir()); err != nil {
		color.Red("Error creating .godspeed directory: %v", err)
		return
	}

	if err := os.WriteFile(servicesFile, updatedData, 0644); err != nil {
		if os.IsPermission(err) {
			action := "link"
			if !add {
				action = "unlink"
			}
			color.Red("Permission denied: Cannot write to services.json")
			color.Yellow("Try running: sudo godspeed %s", action)
		} else {
			color.Red("Error writing services.json: %v", err)
		}
		return
	}

	color.Green("Project data updated successfully.")
}

// IsDockerRunning checks if Docker is running
func IsDockerRunning() bool {
	_, err := ExecuteCommandWithOutput("docker", []string{"version"})
	return err == nil
}

// CheckPrerequisites checks if the necessary prerequisites are available
func CheckPrerequisites() bool {
	if !IsDockerRunning() {
		fmt.Println()
		color.Yellow("godspeed has dependency on docker. Seems like your docker daemon is not running.")
		color.Red("Please run docker daemon first and then try again.")
		fmt.Println()
		return false
	}
	return true
}

// DetectOS returns the current operating system
func DetectOS() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "Mac"
	default:
		return "UNKNOWN"
	}
}
