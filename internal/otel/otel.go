package otel

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/godspeedsystems/godspeed-cli/internal/utils"
)

// Enable enables OpenTelemetry in the project
func Enable() {
	if !utils.IsGodspeedProject() {
		return
	}

	// Check if .env file exists
	envFilePath := filepath.Join(".", ".env")
	if !utils.FileExists(envFilePath) {
		color.Red("Error: .env file not found.")
		return
	}

	// Read .env file
	envContent, err := readEnvFile(envFilePath)
	if err != nil {
		color.Red("Error reading .env file: %v", err)
		return
	}

	// Check if OTEL is already enabled
	if otelEnabled(envContent) {
		color.Yellow("Observability is already enabled in the project.")

		// Install tracing package even if already enabled
		installTracing()
		return
	}

	// Install tracing package
	if err := installTracing(); err != nil {
		color.Red("Error installing tracing package: %v", err)
		return
	}

	// Update .env file
	updatedEnvContent := updateEnvForOtel(envContent, true)
	if err := writeEnvFile(envFilePath, updatedEnvContent); err != nil {
		color.Red("Error updating .env file: %v", err)
		return
	}

	color.Green("Observability has been enabled")
}

// Disable disables OpenTelemetry in the project
func Disable() {
	if !utils.IsGodspeedProject() {
		return
	}

	// Check if .env file exists
	envFilePath := filepath.Join(".", ".env")
	if !utils.FileExists(envFilePath) {
		color.Red("Error: .env file not found.")
		return
	}

	// Read .env file
	envContent, err := readEnvFile(envFilePath)
	if err != nil {
		color.Red("Error reading .env file: %v", err)
		return
	}

	// Check if OTEL is already disabled
	if !otelEnabled(envContent) {
		color.Yellow("Observability is already disabled.")

		// Uninstall tracing package even if already disabled
		uninstallTracing()
		return
	}

	// Uninstall tracing package
	if err := uninstallTracing(); err != nil {
		color.Red("Error uninstalling tracing package: %v", err)
		return
	}

	// Update .env file
	updatedEnvContent := updateEnvForOtel(envContent, false)
	if err := writeEnvFile(envFilePath, updatedEnvContent); err != nil {
		color.Red("Error updating .env file: %v", err)
		return
	}

	color.Green("Observability has been disabled in the project")
}

// readEnvFile reads the content of .env file
func readEnvFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// writeEnvFile writes content to .env file
func writeEnvFile(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}

// otelEnabled checks if OTEL is enabled in the env content
func otelEnabled(envContent []string) bool {
	for _, line := range envContent {
		if strings.TrimSpace(line) == "OTEL_ENABLED=true" {
			return true
		}
	}
	return false
}

// updateEnvForOtel updates the env content for OTEL
func updateEnvForOtel(envContent []string, enable bool) []string {
	// Value to set for OTEL_ENABLED
	otelValue := "true"
	if !enable {
		otelValue = "false"
	}

	// Check if OTEL_ENABLED is already in the file
	for i, line := range envContent {
		if strings.HasPrefix(strings.TrimSpace(line), "OTEL_ENABLED=") {
			envContent[i] = fmt.Sprintf("OTEL_ENABLED=%s", otelValue)
			return envContent
		}
	}

	// If not found, add it
	return append(envContent, fmt.Sprintf("OTEL_ENABLED=%s", otelValue))
}

// installTracing installs the tracing package
func installTracing() error {
	s := utils.NewSpinner("Installing packages... ")
	s.Start()
	defer s.Stop()

	err := utils.ExecuteCommand("npm", []string{
		"install",
		"@godspeedsystems/tracing",
		"--quiet",
		"--no-warnings",
		"--silent",
		"--progress=false",
	})

	if err != nil {
		return err
	}

	fmt.Println("\notel installed successfully!")
	return nil
}

// uninstallTracing uninstalls the tracing package
func uninstallTracing() error {
	s := utils.NewSpinner("Uninstalling packages... ")
	s.Start()
	defer s.Stop()

	err := utils.ExecuteCommand("npm", []string{
		"uninstall",
		"@godspeedsystems/tracing",
		"--quiet",
		"--no-warnings",
		"--silent",
		"--progress=false",
	})

	if err != nil {
		return err
	}

	fmt.Println("\notel uninstalled successfully!")
	return nil
}
