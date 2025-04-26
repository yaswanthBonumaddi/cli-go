package plugin

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
	"gopkg.in/yaml.v3"
)

// Plugin represents a Godspeed plugin
type Plugin struct {
	Value       string `json:"value"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// LoadPluginsList loads the list of available plugins
func LoadPluginsList() ([]Plugin, error) {
	// First try to load from the embedded plugins list
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	// Path to the plugins list relative to the executable
	pluginsPath := filepath.Join(filepath.Dir(execPath), "assets", "plugins_list.json")

	// Try to read the plugins list
	data, err := ioutil.ReadFile(pluginsPath)
	if err != nil {
		// If not found, try to search on npm
		return searchPluginsFromNpm()
	}

	var plugins []Plugin
	err = json.Unmarshal(data, &plugins)
	if err != nil {
		return nil, err
	}

	return plugins, nil
}

// searchPluginsFromNpm searches for Godspeed plugins on npm
func searchPluginsFromNpm() ([]Plugin, error) {
	cmd := exec.Command("npm", "search", "@godspeedsystems/plugins", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var searchResults []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
	}

	err = json.Unmarshal(output, &searchResults)
	if err != nil {
		return nil, err
	}

	// Convert npm search results to Plugin format
	var plugins []Plugin
	for _, result := range searchResults {
		// Extract the plugin name without the prefix
		nameParts := strings.Split(result.Name, "plugins-")
		if len(nameParts) > 1 {
			plugins = append(plugins, Plugin{
				Value:       result.Name,
				Name:        nameParts[1],
				Description: result.Description,
			})
		}
	}

	return plugins, nil
}

// GetInstalledPlugins returns a list of installed plugins
func GetInstalledPlugins() (map[string]string, error) {
	if !utils.IsGodspeedProject() {
		return nil, fmt.Errorf("not a godspeed project")
	}

	// Read package.json
	pkgPath := filepath.Join(".", "package.json")
	if !utils.FileExists(pkgPath) {
		return nil, fmt.Errorf("package.json not found")
	}

	pkgData, err := ioutil.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}

	err = json.Unmarshal(pkgData, &pkg)
	if err != nil {
		return nil, err
	}

	// Filter Godspeed plugins
	plugins := make(map[string]string)
	for name, version := range pkg.Dependencies {
		if strings.HasPrefix(name, "@godspeedsystems/plugins") {
			plugins[name] = version
		}
	}

	return plugins, nil
}

// Add adds a plugin to the project
func Add(pluginName string) {
	if !utils.IsGodspeedProject() {
		return
	}

	// Load available plugins
	availablePlugins, err := LoadPluginsList()
	if err != nil {
		color.Red("Error loading plugins list: %v", err)
		return
	}

	// Get installed plugins
	installedPlugins, err := GetInstalledPlugins()
	if err != nil {
		color.Red("Error checking installed plugins: %v", err)
		return
	}

	// Filter out installed plugins
	var missingPlugins []Plugin
	for _, plugin := range availablePlugins {
		if _, installed := installedPlugins[plugin.Value]; !installed {
			missingPlugins = append(missingPlugins, plugin)
		}
	}

	// No plugin name provided, show interactive menu
	if pluginName == "" {
		if len(missingPlugins) == 0 {
			color.Red("All plugins are already installed.")
			return
		}

		var selectedPlugins []string
		options := make([]string, len(missingPlugins))
		optionsMap := make(map[string]string)

		for i, plugin := range missingPlugins {
			displayName := fmt.Sprintf("%s - %s", plugin.Name, plugin.Description)
			options[i] = displayName
			optionsMap[displayName] = plugin.Value
		}

		prompt := &survey.MultiSelect{
			Message: "Please select godspeed plugin to install:",
			Options: options,
		}

		err = survey.AskOne(prompt, &selectedPlugins)
		if err != nil {
			color.Red("Error: %v", err)
			return
		}

		if len(selectedPlugins) == 0 {
			color.Red("No plugins selected.")
			return
		}

		// Convert display names to plugin names
		pluginsToInstall := make([]string, len(selectedPlugins))
		for i, name := range selectedPlugins {
			pluginsToInstall[i] = optionsMap[name]
		}

		installPlugins(pluginsToInstall)
	} else {
		// Find the plugin by name
		found := false
		for _, plugin := range availablePlugins {
			if plugin.Value == pluginName {
				found = true
				break
			}
		}

		if !found {
			color.Red("\nPlease provide a valid plugin name.\n")
			return
		}

		// Check if plugin is already installed
		if _, installed := installedPlugins[pluginName]; installed {
			color.Yellow("Plugin %s is already installed.", pluginName)
			return
		}

		installPlugins([]string{pluginName})

		color.Cyan("\nFor detailed documentation and examples, visit:")
		color.Yellow("https://www.npmjs.com/package/%s\n", pluginName)
	}
}

// Remove removes a plugin from the project
func Remove(pluginName string) {
	if !utils.IsGodspeedProject() {
		return
	}

	// Get installed plugins
	installedPlugins, err := GetInstalledPlugins()
	if err != nil {
		color.Red("Error checking installed plugins: %v", err)
		return
	}

	if len(installedPlugins) == 0 {
		color.Red("There are no eventsource/datasource plugins installed.")
		return
	}

	var pluginsToRemove []string

	// If plugin name is provided, remove that specific plugin
	if pluginName != "" {
		if _, installed := installedPlugins[pluginName]; !installed {
			color.Red("Plugin %s is not installed.", pluginName)
			return
		}
		pluginsToRemove = []string{pluginName}
	} else {
		// Interactive selection
		var selectedPlugins []string
		options := make([]string, 0, len(installedPlugins))
		optionsMap := make(map[string]string)

		// Load available plugins to get descriptions
		availablePlugins, err := LoadPluginsList()
		if err != nil {
			color.Red("Error loading plugins list: %v", err)
			return
		}

		// Create a map for quick lookup
		pluginDescriptions := make(map[string]string)
		for _, plugin := range availablePlugins {
			pluginDescriptions[plugin.Value] = plugin.Description
		}

		// Create options with descriptions
		for name := range installedPlugins {
			description := pluginDescriptions[name]
			if description == "" {
				description = "No description available"
			}
			displayName := fmt.Sprintf("%s - %s", name, description)
			options = append(options, displayName)
			optionsMap[displayName] = name
		}

		prompt := &survey.MultiSelect{
			Message: "Please select godspeed plugin to uninstall:",
			Options: options,
		}

		err = survey.AskOne(prompt, &selectedPlugins)
		if err != nil {
			color.Red("Error: %v", err)
			return
		}

		if len(selectedPlugins) == 0 {
			color.Red("No plugins selected to remove.")
			return
		}

		// Convert display names to plugin names
		for _, name := range selectedPlugins {
			pluginsToRemove = append(pluginsToRemove, optionsMap[name])
		}
	}

	removePlugins(pluginsToRemove)
}

// Update updates plugins in the project
func Update() {
	if !utils.IsGodspeedProject() {
		return
	}

	// Get installed plugins
	installedPlugins, err := GetInstalledPlugins()
	if err != nil {
		color.Red("Error checking installed plugins: %v", err)
		return
	}

	if len(installedPlugins) == 0 {
		color.Red("There are no eventsource/datasource plugins installed.")
		return
	}

	// Interactive selection
	var selectedPlugins []string
	options := make([]string, 0, len(installedPlugins))
	optionsMap := make(map[string]string)

	// Load available plugins to get descriptions
	availablePlugins, err := LoadPluginsList()
	if err != nil {
		color.Red("Error loading plugins list: %v", err)
		return
	}

	// Create a map for quick lookup
	pluginDescriptions := make(map[string]string)
	for _, plugin := range availablePlugins {
		pluginDescriptions[plugin.Value] = plugin.Description
	}

	// Create options with descriptions
	for name := range installedPlugins {
		description := pluginDescriptions[name]
		if description == "" {
			description = "No description available"
		}
		displayName := fmt.Sprintf("%s - %s", name, description)
		options = append(options, displayName)
		optionsMap[displayName] = name
	}

	prompt := &survey.MultiSelect{
		Message: "Please select godspeed plugin to update:",
		Options: options,
	}

	err = survey.AskOne(prompt, &selectedPlugins)
	if err != nil {
		color.Red("Error: %v", err)
		return
	}

	if len(selectedPlugins) == 0 {
		color.Red("No plugins selected to update.")
		return
	}

	// Convert display names to plugin names
	var pluginsToUpdate []string
	for _, name := range selectedPlugins {
		pluginsToUpdate = append(pluginsToUpdate, optionsMap[name])
	}

	updatePlugins(pluginsToUpdate)
}

// installPlugins installs the specified plugins
func installPlugins(plugins []string) {
	if len(plugins) == 0 {
		return
	}

	// Start spinner
	s := utils.NewSpinner("Installing plugins... ")
	s.Start()

	// Create npm install command with all plugins
	args := append([]string{"install"}, plugins...)
	args = append(args, "--quiet", "--no-warnings", "--silent", "--progress=false")

	cmd := exec.Command("npm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	s.Stop()

	if err != nil {
		color.Red("\nError installing plugins: %v", err)
		return
	}

	color.Green("\nPlugins installed successfully!")

	// Create necessary files for each plugin
	for _, pluginName := range plugins {
		if err := createPluginFiles(pluginName); err != nil {
			color.Red("Error creating files for %s: %v", pluginName, err)
		}
	}

	color.Cyan("Happy coding with Godspeed! 🚀🎉\n")
}

// removePlugins removes the specified plugins
func removePlugins(plugins []string) {
	if len(plugins) == 0 {
		return
	}

	// For each plugin, remove the associated files
	for _, pluginName := range plugins {
		if err := removePluginFiles(pluginName); err != nil {
			color.Red("Error removing files for %s: %v", pluginName, err)
		}
	}

	// Start spinner
	s := utils.NewSpinner("Uninstalling plugins... ")
	s.Start()

	// Create npm uninstall command with all plugins
	args := append([]string{"uninstall"}, plugins...)
	args = append(args, "--quiet", "--no-warnings", "--silent", "--progress=false")

	cmd := exec.Command("npm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	s.Stop()

	if err != nil {
		color.Red("\nError uninstalling plugins: %v", err)
		return
	}

	color.Green("\nPlugins uninstalled successfully!")
	color.Cyan("Happy coding with Godspeed! 🚀🎉\n")
}

// updatePlugins updates the specified plugins
func updatePlugins(plugins []string) {
	if len(plugins) == 0 {
		return
	}

	// Start spinner
	s := utils.NewSpinner("Updating plugins... ")
	s.Start()

	// Create npm update command with all plugins
	args := append([]string{"update"}, plugins...)
	args = append(args, "--quiet", "--no-warnings", "--silent", "--progress=false")

	cmd := exec.Command("npm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	s.Stop()

	if err != nil {
		color.Red("\nError updating plugins: %v", err)
		return
	}

	color.Green("\nPlugins updated successfully!")
	color.Cyan("Happy coding with Godspeed! 🚀🎉\n")
}

// Module types constants
const (
	ModuleTypeDS   = "DS"
	ModuleTypeES   = "ES"
	ModuleTypeBoth = "BOTH"
)

// getModuleInfo gets information about a plugin module
func getModuleInfo(pluginName string) (moduleType, loaderFileName, yamlFileName string, defaultConfig map[string]interface{}, err error) {
	// Run a small Node.js script to get the module info
	script := fmt.Sprintf(`
		try {
			const Module = require('%s');
			console.log(JSON.stringify({
				moduleType: Module.SourceType,
				loaderFileName: Module.Type,
				yamlFileName: Module.CONFIG_FILE_NAME,
				defaultConfig: Module.DEFAULT_CONFIG || {}
			}));
		} catch (e) {
			console.error(e.message);
			process.exit(1);
		}
	`, pluginName)

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", "", "", nil, fmt.Errorf("error getting module info: %v", err)
	}

	var result struct {
		ModuleType     string                 `json:"moduleType"`
		LoaderFileName string                 `json:"loaderFileName"`
		YamlFileName   string                 `json:"yamlFileName"`
		DefaultConfig  map[string]interface{} `json:"defaultConfig"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", "", "", nil, fmt.Errorf("error parsing module info: %v", err)
	}

	return result.ModuleType, result.LoaderFileName, result.YamlFileName, result.DefaultConfig, nil
}

// createPluginFiles creates the necessary files for a plugin
func createPluginFiles(pluginName string) error {
	moduleType, loaderFileName, yamlFileName, defaultConfig, err := getModuleInfo(pluginName)
	if err != nil {
		return err
	}

	switch moduleType {
	case ModuleTypeBoth:
		// Create files for both EventSource and DataSource
		if err := createEventSourceFiles(pluginName, loaderFileName, yamlFileName, defaultConfig); err != nil {
			return err
		}

		if err := createDataSourceFiles(pluginName, loaderFileName, yamlFileName, defaultConfig); err != nil {
			return err
		}

	case ModuleTypeDS:
		// Create files for DataSource
		if err := createDataSourceFiles(pluginName, loaderFileName, yamlFileName, defaultConfig); err != nil {
			return err
		}

	case ModuleTypeES:
		// Create files for EventSource
		if err := createEventSourceFiles(pluginName, loaderFileName, yamlFileName, defaultConfig); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown module type: %s", moduleType)
	}

	return nil
}

// createEventSourceFiles creates the files for an EventSource plugin
func createEventSourceFiles(pluginName, loaderFileName, yamlFileName string, defaultConfig map[string]interface{}) error {
	// Create types directory if it doesn't exist
	typesDir := filepath.Join("src", "eventsources", "types")
	if err := utils.CreateDir(typesDir); err != nil {
		return err
	}

	// Create TypeScript file
	tsContent := fmt.Sprintf(`
import { EventSource } from '%s';
export default EventSource;
	`, pluginName)

	tsPath := filepath.Join(typesDir, fmt.Sprintf("%s.ts", loaderFileName))
	if err := ioutil.WriteFile(tsPath, []byte(tsContent), 0644); err != nil {
		return err
	}

	// Create YAML file
	config := map[string]interface{}{
		"type": loaderFileName,
	}

	// Add default config
	for k, v := range defaultConfig {
		config[k] = v
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	yamlPath := filepath.Join("src", "eventsources", fmt.Sprintf("%s.yaml", yamlFileName))
	return ioutil.WriteFile(yamlPath, yamlData, 0644)
}

// createDataSourceFiles creates the files for a DataSource plugin
func createDataSourceFiles(pluginName, loaderFileName, yamlFileName string, defaultConfig map[string]interface{}) error {
	// Create types directory if it doesn't exist
	typesDir := filepath.Join("src", "datasources", "types")
	if err := utils.CreateDir(typesDir); err != nil {
		return err
	}

	// Create TypeScript file
	tsContent := fmt.Sprintf(`
import { DataSource } from '%s';
export default DataSource;
	`, pluginName)

	tsPath := filepath.Join(typesDir, fmt.Sprintf("%s.ts", loaderFileName))
	if err := ioutil.WriteFile(tsPath, []byte(tsContent), 0644); err != nil {
		return err
	}

	// Skip YAML file for prisma
	if loaderFileName == "prisma" {
		return nil
	}

	// Create YAML file
	config := map[string]interface{}{
		"type": loaderFileName,
	}

	// Add default config
	for k, v := range defaultConfig {
		config[k] = v
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	yamlPath := filepath.Join("src", "datasources", fmt.Sprintf("%s.yaml", yamlFileName))
	return ioutil.WriteFile(yamlPath, yamlData, 0644)
}

// removePluginFiles removes the files associated with a plugin
func removePluginFiles(pluginName string) error {
	moduleType, loaderFileName, yamlFileName, _, err := getModuleInfo(pluginName)
	if err != nil {
		return err
	}

	switch moduleType {
	case ModuleTypeBoth:
		// Remove both EventSource and DataSource files
		if err := removeEventSourceFiles(loaderFileName, yamlFileName); err != nil {
			return err
		}

		if err := removeDataSourceFiles(loaderFileName, yamlFileName); err != nil {
			return err
		}

	case ModuleTypeDS:
		// Remove DataSource files
		if err := removeDataSourceFiles(loaderFileName, yamlFileName); err != nil {
			return err
		}

	case ModuleTypeES:
		// Remove EventSource files
		if err := removeEventSourceFiles(loaderFileName, yamlFileName); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown module type: %s", moduleType)
	}

	return nil
}

// removeEventSourceFiles removes the files for an EventSource plugin
func removeEventSourceFiles(loaderFileName, yamlFileName string) error {
	// Remove TypeScript file
	tsPath := filepath.Join("src", "eventsources", "types", fmt.Sprintf("%s.ts", loaderFileName))
	if utils.FileExists(tsPath) {
		if err := os.Remove(tsPath); err != nil {
			return err
		}
	}

	// Remove YAML file
	yamlPath := filepath.Join("src", "eventsources", fmt.Sprintf("%s.yaml", yamlFileName))
	if utils.FileExists(yamlPath) {
		if err := os.Remove(yamlPath); err != nil {
			return err
		}
	}

	return nil
}

// removeDataSourceFiles removes the files for a DataSource plugin
func removeDataSourceFiles(loaderFileName, yamlFileName string) error {
	// Remove TypeScript file
	tsPath := filepath.Join("src", "datasources", "types", fmt.Sprintf("%s.ts", loaderFileName))
	if utils.FileExists(tsPath) {
		if err := os.Remove(tsPath); err != nil {
			return err
		}
	}

	// Skip YAML file for prisma
	if loaderFileName == "prisma" {
		return nil
	}

	// Remove YAML file
	yamlPath := filepath.Join("src", "datasources", fmt.Sprintf("%s.yaml", yamlFileName))
	if utils.FileExists(yamlPath) {
		if err := os.Remove(yamlPath); err != nil {
			return err
		}
	}

	return nil
}
