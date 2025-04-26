package devops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/godspeedsystems/godspeed-cli/internal/utils"
)

// DevopsPlugin represents a devops plugin with metadata
type DevopsPlugin struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// Install installs a devops plugin
func Install(pluginName string) {
	gsDevopsPluginsDir := filepath.Join(utils.UserHomeDir(), ".godspeed", "devops-plugins")

	// Create plugins directory if it doesn't exist
	if err := utils.CreateDir(gsDevopsPluginsDir); err != nil {
		color.Red("Error creating plugins directory: %v", err)
		return
	}

	// Initialize package.json if it doesn't exist
	packageJsonPath := filepath.Join(gsDevopsPluginsDir, "package.json")
	if !utils.FileExists(packageJsonPath) {
		cmd := exec.Command("npm", "init", "--yes")
		cmd.Dir = gsDevopsPluginsDir
		if err := cmd.Run(); err != nil {
			color.Red("Error initializing package.json: %v", err)
			return
		}
	}

	// If no plugin name provided, show interactive selection
	if pluginName == "" {
		availablePlugins, err := searchDevopsPlugins()
		if err != nil {
			color.Red("Error searching for devops plugins: %v", err)
			return
		}

		if len(availablePlugins) == 0 {
			color.Red("No devops plugins found.")
			return
		}

		options := make([]string, len(availablePlugins))
		for i, plugin := range availablePlugins {
			options[i] = fmt.Sprintf("%s - %s", plugin.Name, plugin.Description)
		}

		var selected string
		prompt := &survey.Select{
			Message: "Please select devops plugin to install:",
			Options: options,
		}

		if err := survey.AskOne(prompt, &selected); err != nil {
			color.Red("Error: %v", err)
			return
		}

		// Extract plugin name from the selected option
		parts := strings.SplitN(selected, " - ", 2)
		pluginName = parts[0]
	}

	// Install the plugin
	color.Yellow("Installing %s...", pluginName)
	cmd := exec.Command("npm", "install", pluginName)
	cmd.Dir = gsDevopsPluginsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		color.Red("Error installing plugin: %v", err)
		return
	}

	color.Green("Successfully installed %s", pluginName)
}

// Remove removes a devops plugin
func Remove(pluginName string) {
	gsDevopsPluginsDir := filepath.Join(utils.UserHomeDir(), ".godspeed", "devops-plugins")

	// Check if plugins directory exists
	if !utils.DirExists(gsDevopsPluginsDir) {
		color.Red("Devops plugins directory not found.")
		return
	}

	// Check if package.json exists
	packageJsonPath := filepath.Join(gsDevopsPluginsDir, "package.json")
	if !utils.FileExists(packageJsonPath) {
		color.Red("No devops plugins are installed.")
		return
	}

	// Read package.json to get installed plugins
	data, err := ioutil.ReadFile(packageJsonPath)
	if err != nil {
		color.Red("Error reading package.json: %v", err)
		return
	}

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		color.Red("Error parsing package.json: %v", err)
		return
	}

	if pkg.Dependencies == nil || len(pkg.Dependencies) == 0 {
		color.Red("No devops plugins are installed.")
		return
	}

	// If no plugin name provided, show interactive selection
	if pluginName == "" {
		options := make([]string, 0, len(pkg.Dependencies))
		for name := range pkg.Dependencies {
			options = append(options, name)
		}

		var selected string
		prompt := &survey.Select{
			Message: "Please select devops plugin to remove:",
			Options: options,
		}

		if err := survey.AskOne(prompt, &selected); err != nil {
			color.Red("Error: %v", err)
			return
		}

		pluginName = selected
	} else {
		// Check if the specified plugin is installed
		if _, ok := pkg.Dependencies[pluginName]; !ok {
			color.Red("Plugin %s is not installed.", pluginName)
			return
		}
	}

	// Remove the plugin
	color.Yellow("Removing %s...", pluginName)
	cmd := exec.Command("npm", "uninstall", pluginName)
	cmd.Dir = gsDevopsPluginsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		color.Red("Error removing plugin: %v", err)
		return
	}

	color.Green("Successfully removed %s", pluginName)
}

// Update updates a devops plugin
func Update() {
	gsDevopsPluginsDir := filepath.Join(utils.UserHomeDir(), ".godspeed", "devops-plugins")

	// Check if plugins directory exists
	if !utils.DirExists(gsDevopsPluginsDir) {
		color.Red("Devops plugins directory not found.")
		return
	}

	// Check if package.json exists
	packageJsonPath := filepath.Join(gsDevopsPluginsDir, "package.json")
	if !utils.FileExists(packageJsonPath) {
		color.Red("No devops plugins are installed.")
		return
	}

	// Read package.json to get installed plugins
	data, err := ioutil.ReadFile(packageJsonPath)
	if err != nil {
		color.Red("Error reading package.json: %v", err)
		return
	}

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		color.Red("Error parsing package.json: %v", err)
		return
	}

	if pkg.Dependencies == nil || len(pkg.Dependencies) == 0 {
		color.Red("No devops plugins are installed.")
		return
	}

	// Show interactive selection
	options := make([]string, 0, len(pkg.Dependencies))
	for name := range pkg.Dependencies {
		options = append(options, name)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Please select devops plugin to update:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		color.Red("Error: %v", err)
		return
	}

	// Update the plugin
	color.Yellow("Updating %s...", selected)
	cmd := exec.Command("npm", "install", fmt.Sprintf("%s@latest", selected))
	cmd.Dir = gsDevopsPluginsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		color.Red("Error updating plugin: %v", err)
		return
	}

	color.Green("Successfully updated %s", selected)
}

// List lists available or installed devops plugins
func List(installed bool) {
	if installed {
		listInstalledPlugins()
	} else {
		listAvailablePlugins()
	}
}

// listInstalledPlugins lists installed devops plugins
func listInstalledPlugins() {
	gsDevopsPluginsDir := filepath.Join(utils.UserHomeDir(), ".godspeed", "devops-plugins")

	// Check if plugins directory exists
	if !utils.DirExists(gsDevopsPluginsDir) {
		color.Red("Devops plugins directory not found.")
		return
	}

	// Check if package.json exists
	packageJsonPath := filepath.Join(gsDevopsPluginsDir, "package.json")
	if !utils.FileExists(packageJsonPath) {
		color.Red("No devops-plugin is installed")
		return
	}

	// Read package.json to get installed plugins
	data, err := ioutil.ReadFile(packageJsonPath)
	if err != nil {
		color.Red("Error reading package.json: %v", err)
		return
	}

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		color.Red("Error parsing package.json: %v", err)
		return
	}

	if pkg.Dependencies == nil || len(pkg.Dependencies) == 0 {
		color.Red("There are no devops plugins installed.")
		return
	}

	for name := range pkg.Dependencies {
		fmt.Printf("-> %s\n", name)
	}
}

// listAvailablePlugins lists available devops plugins
func listAvailablePlugins() {
	plugins, err := searchDevopsPlugins()
	if err != nil {
		color.Red("Error searching for devops plugins: %v", err)
		return
	}

	if len(plugins) == 0 {
		color.Red("No devops plugins found.")
		return
	}

	fmt.Println("List of available devops plugins:")
	for _, plugin := range plugins {
		fmt.Printf("-> %s\n", plugin.Name)
	}
}

// searchDevopsPlugins searches for available devops plugins on npm
func searchDevopsPlugins() ([]DevopsPlugin, error) {
	cmd := exec.Command("npm", "search", "@godspeedsystems/devops-plugin", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var searchResults []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
	}

	if err := json.Unmarshal(output, &searchResults); err != nil {
		return nil, err
	}

	plugins := make([]DevopsPlugin, len(searchResults))
	for i, result := range searchResults {
		plugins[i] = DevopsPlugin{
			Name:        result.Name,
			Description: result.Description,
			Version:     result.Version,
		}
	}

	return plugins, nil
}
