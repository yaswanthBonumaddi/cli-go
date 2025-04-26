package create

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing" // Add this line
	"github.com/godspeedsystems/godspeed-cli/internal/utils"
)

// GodspeedOptions represents the configuration for a godspeed project
type GodspeedOptions struct {
	ProjectName          string                 `json:"projectName"`
	GSNodeServiceVersion string                 `json:"gsNodeServiceVersion"`
	ServicePort          int                    `json:"servicePort"`
	MongoDB              interface{}            `json:"mongodb"`
	PostgreSQL           interface{}            `json:"postgresql"`
	MySQL                interface{}            `json:"mysql"`
	Kafka                interface{}            `json:"kafka"`
	Elasticsearch        interface{}            `json:"elasticsearch"`
	Redis                interface{}            `json:"redis"`
	UserUID              int                    `json:"userUID"`
	Meta                 map[string]interface{} `json:"meta"`
}

// Execute creates a new godspeed project
func Execute(projectName, fromTemplate, fromExample, cliVersion string) {
	fmt.Println()

	// Create project directory
	projectDirPath := filepath.Join(".", projectName)

	// Validate and create project directory
	if err := validateAndCreateProjectDirectory(projectDirPath); err != nil {
		color.Red("Error creating project directory: %v", err)
		os.Exit(1)
	}

	var godspeedOptions *GodspeedOptions

	// Handle template or clone default template
	if fromTemplate != "" {
		if err := copyingLocalTemplate(projectDirPath, fromTemplate); err != nil {
			color.Red("Error copying template: %v", err)
			os.Exit(1)
		}
	} else {
		if err := cloneProjectTemplate(projectDirPath); err != nil {
			color.Red("Error cloning template: %v", err)
			os.Exit(1)
		}
	}

	// Generate from examples
	var err error
	if godspeedOptions, err = generateFromExamples(projectDirPath, fromExample); err != nil {
		color.Red("Error generating from examples: %v", err)
		os.Exit(1)
	}

	// If no options were loaded from examples, use interactive mode
	if godspeedOptions == nil {
		godspeedOptions, err = interactiveMode(projectName)
		if err != nil {
			color.Red("Error in interactive mode: %v", err)
			os.Exit(1)
		}
	}

	// Set project name and metadata
	timestamp := getCurrentTimestamp()
	godspeedOptions.ProjectName = projectName
	godspeedOptions.Meta = map[string]interface{}{
		"createTimestamp":           timestamp,
		"lastUpdateTimestamp":       timestamp,
		"cliVersionWhileCreation":   cliVersion,
		"cliVersionWhileLastUpdate": cliVersion,
	}

	// Generate project files
	if err := generateProjectFromDotGodspeed(projectName, projectDirPath, godspeedOptions, fromExample); err != nil {
		color.Red("Error generating project: %v", err)
		utils.RemoveDir(projectDirPath)
		os.Exit(1)
	}

	// Install specific plugins for examples
	if fromExample == "mongo-as-prisma" {
		spinner := utils.NewSpinner("Installing prisma plugin... ")
		spinner.Start()
		utils.ExecuteCommand("npm", []string{"install", "@godspeedsystems/plugins-prisma-as-datastore", "--quiet"})
		spinner.Stop()
	}

	// Install dependencies
	if err := installDependencies(projectDirPath, projectName); err != nil {
		color.Red("Error installing dependencies: %v", err)
		os.Exit(1)
	}

	color.Green("\nSuccessfully created the project %s.", color.YellowString(projectName))
	color.Green("Use `godspeed help` command for available commands.")
	fmt.Println()
	color.Green("\nHappy building microservices with Godspeed! 🚀🎉\n")
}

// validateAndCreateProjectDirectory ensures the project directory can be created
func validateAndCreateProjectDirectory(projectDirPath string) error {
	// Check if directory already exists
	if utils.DirExists(projectDirPath) {
		var overwrite bool
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("%s already exists.\nDo you want to overwrite the project folder?", color.YellowString(projectDirPath)),
			Default: false,
		}

		if err := survey.AskOne(prompt, &overwrite); err != nil {
			return err
		}

		if !overwrite {
			fmt.Println(color.RedString("\nExiting godspeed create without creating project."))
			os.Exit(0)
		}

		// Remove existing directory
		if err := utils.RemoveDir(projectDirPath); err != nil {
			return err
		}
	}

	// Create project directory
	return os.MkdirAll(projectDirPath, 0755)
}

// cloneProjectTemplate clones the godspeed template repository
func cloneProjectTemplate(projectDirPath string) error {
	color.Yellow("Cloning project template from %s branch %s to %s",
		os.Getenv("GITHUB_REPO_URL"),
		os.Getenv("GITHUB_REPO_BRANCH"),
		projectDirPath)

	// Ensure the directory exists
	if err := os.MkdirAll(projectDirPath, 0755); err != nil {
		return fmt.Errorf("error creating project directory: %v", err)
	}

	repoURL := os.Getenv("GITHUB_REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/godspeedsystems/godspeed-scaffolding.git"
		color.Yellow("Using default repo URL: %s", repoURL)
	}

	branch := os.Getenv("GITHUB_REPO_BRANCH")
	if branch == "" {
		branch = "template"
		color.Yellow("Using default branch: %s", branch)
	}

	// Try cloning with go-git
	color.Yellow("Attempting to clone using go-git...")
	_, err := git.PlainClone(projectDirPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Depth:         1,
		Progress:      os.Stdout, // Show progress
	})

	if err != nil {
		color.Red("go-git clone failed: %v", err)

		// Fallback to system git command
		color.Yellow("Falling back to system git command...")
		cmd := exec.Command("git", "clone", repoURL, "--branch", branch, "--depth", "1", projectDirPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %v\nOutput: %s", err, output)
		}
		color.Green("Git clone successful using system git")
	} else {
		color.Green("Git clone successful using go-git")
	}

	// Verify the .template directory exists
	templateDir := filepath.Join(projectDirPath, ".template")
	if !utils.DirExists(templateDir) {
		return fmt.Errorf(".template directory not found after cloning. Repository structure may be incorrect")
	}

	// Remove .git directory to start fresh
	if err := utils.RemoveDir(filepath.Join(projectDirPath, ".git")); err != nil {
		return err
	}

	color.Green("Cloning template successful.")
	return nil
}

// copyingLocalTemplate copies a local template to project directory
func copyingLocalTemplate(projectDirPath, templateDir string) error {
	if !utils.DirExists(templateDir) {
		return fmt.Errorf("%s does not exist or path is incorrect", color.RedString(templateDir))
	}

	color.Yellow("Copying template from %s", color.YellowString(templateDir))
	if err := utils.CopyDir(templateDir, projectDirPath); err != nil {
		return err
	}
	color.Green("Copying template successful.")
	return nil
}

// generateFromExamples generates project from examples
func generateFromExamples(projectDirPath, exampleName string) (*GodspeedOptions, error) {
	if exampleName == "" {
		exampleName = "hello-world"
	}

	color.Yellow("Generating project with %s examples.", color.YellowString(exampleName))

	examplesPath := filepath.Join(projectDirPath, ".template", "examples", exampleName)
	if !utils.DirExists(examplesPath) {
		return nil, fmt.Errorf("%s is not a valid example", color.RedString(exampleName))
	}

	// Copy example files to project directory
	if err := utils.CopyDir(examplesPath, projectDirPath); err != nil {
		return nil, err
	}

	// Check if there's a .godspeed file from the example
	godspeedFilePath := filepath.Join(examplesPath, ".godspeed")
	if utils.FileExists(godspeedFilePath) {
		return readDotGodspeed(projectDirPath)
	}

	return nil, nil
}

// readDotGodspeed reads .godspeed configuration file
func readDotGodspeed(projectDirPath string) (*GodspeedOptions, error) {
	data, err := os.ReadFile(filepath.Join(projectDirPath, ".godspeed"))
	if err != nil {
		return nil, err
	}

	var options GodspeedOptions
	if err := json.Unmarshal(data, &options); err != nil {
		return nil, err
	}

	return &options, nil
}

// interactiveMode prompts user for project configuration
func interactiveMode(projectName string) (*GodspeedOptions, error) {
	fmt.Println()

	versions, err := fetchFrameworkVersionTags()
	if err != nil {
		return nil, err
	}

	// MongoDB questions
	var useMongoDB bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Do you want mongoDB as database?",
		Default: false,
	}, &useMongoDB); err != nil {
		return nil, err
	}

	var mongoDBOptions map[string]interface{}
	if useMongoDB {
		var dbName string
		var port1, port2, port3 int

		if err := survey.AskOne(&survey.Input{
			Message: "What do you want to name your MongoDB database?",
			Default: "godspeed",
		}, &dbName, survey.WithValidator(wordValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter the port for MongoDB node[1].",
			Default: "27017",
		}, &port1, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter the port for MongoDB node[2].",
			Default: "27018",
		}, &port2, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter the port for MongoDB node[3].",
			Default: "27019",
		}, &port3, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		mongoDBOptions = map[string]interface{}{
			"dbName": dbName,
			"ports":  []int{port1, port2, port3},
		}
	}

	// MySQL questions
	var useMySQL bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Do you want to use MySQL as database?",
		Default: false,
	}, &useMySQL); err != nil {
		return nil, err
	}

	var mysqlOptions map[string]interface{}
	if useMySQL {
		var dbName string
		var port int

		if err := survey.AskOne(&survey.Input{
			Message: "What will be the name of MySQL database?",
			Default: "godspeed",
		}, &dbName, survey.WithValidator(wordValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "What will be the port of MySQL database?",
			Default: "3306",
		}, &port, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		mysqlOptions = map[string]interface{}{
			"dbName": dbName,
			"port":   port,
		}
	}

	// PostgreSQL questions
	var usePostgreSQL bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Do you want to use PostgreSQL as database?",
		Default: false,
	}, &usePostgreSQL); err != nil {
		return nil, err
	}

	var postgresqlOptions map[string]interface{}
	if usePostgreSQL {
		var dbName string
		var port int

		if err := survey.AskOne(&survey.Input{
			Message: "What will be the name of PostgreSQL database?",
			Default: "godspeed",
		}, &dbName, survey.WithValidator(wordValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "What will be the port of PostgreSQL database?",
			Default: "5432",
		}, &port, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		postgresqlOptions = map[string]interface{}{
			"dbName": dbName,
			"port":   port,
		}
	}

	// Kafka questions
	var useKafka bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Do you want to use Apache Kafka?",
		Default: false,
	}, &useKafka); err != nil {
		return nil, err
	}

	var kafkaOptions map[string]interface{}
	if useKafka {
		var kafkaPort, zookeeperPort int

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter kafka port.",
			Default: "9092",
		}, &kafkaPort, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter zookeeper port.",
			Default: "2181",
		}, &zookeeperPort, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		kafkaOptions = map[string]interface{}{
			"kafkaPort":     kafkaPort,
			"zookeeperPort": zookeeperPort,
		}
	}

	// Elasticsearch questions
	var useElasticsearch bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Do you want to use Elasticsearch?",
		Default: false,
	}, &useElasticsearch); err != nil {
		return nil, err
	}

	var elasticsearchOptions map[string]interface{}
	if useElasticsearch {
		var port int

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter Elasticsearch port.",
			Default: "9200",
		}, &port, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		elasticsearchOptions = map[string]interface{}{
			"port": port,
		}
	}

	// Redis questions
	var useRedis bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Do you want to use Redis as database?",
		Default: false,
	}, &useRedis); err != nil {
		return nil, err
	}

	var redisOptions map[string]interface{}
	if useRedis {
		var dbName string
		var port int

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter Redis database name.",
			Default: "godspeed",
		}, &dbName, survey.WithValidator(wordValidator)); err != nil {
			return nil, err
		}

		if err := survey.AskOne(&survey.Input{
			Message: "Please enter the Redis port?",
			Default: "6379",
		}, &port, survey.WithValidator(portValidator)); err != nil {
			return nil, err
		}

		redisOptions = map[string]interface{}{
			"dbName": dbName,
			"port":   port,
		}
	}

	// Service port
	var servicePort int
	if err := survey.AskOne(&survey.Input{
		Message: "Please enter host port on which you want to run your service.",
		Default: "3000",
	}, &servicePort, survey.WithValidator(portValidator)); err != nil {
		return nil, err
	}

	// Framework version
	var gsNodeServiceVersion string
	if err := survey.AskOne(&survey.Select{
		Message: "Please select gs-node-service(Godspeed Framework) version.",
		Options: versions,
		Default: "latest",
	}, &gsNodeServiceVersion); err != nil {
		return nil, err
	}

	fmt.Println()

	// Set MongoDB to false if not used
	var mongoDB interface{} = false
	if useMongoDB {
		mongoDB = mongoDBOptions
	}

	// Set MySQL to false if not used
	var mysql interface{} = false
	if useMySQL {
		mysql = mysqlOptions
	}

	// Set PostgreSQL to false if not used
	var postgresql interface{} = false
	if usePostgreSQL {
		postgresql = postgresqlOptions
	}

	// Set Kafka to false if not used
	var kafka interface{} = false
	if useKafka {
		kafka = kafkaOptions
	}

	// Set Elasticsearch to false if not used
	var elasticsearch interface{} = false
	if useElasticsearch {
		elasticsearch = elasticsearchOptions
	}

	// Set Redis to false if not used
	var redis interface{} = false
	if useRedis {
		redis = redisOptions
	}

	return &GodspeedOptions{
		ProjectName:          projectName,
		GSNodeServiceVersion: gsNodeServiceVersion,
		ServicePort:          servicePort,
		MongoDB:              mongoDB,
		MySQL:                mysql,
		PostgreSQL:           postgresql,
		Kafka:                kafka,
		Elasticsearch:        elasticsearch,
		Redis:                redis,
		UserUID:              getUserID(),
	}, nil
}

// getCurrentTimestamp returns the current time in ISO 8601 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// getUserID gets the current user ID
func getUserID() int {
	if runtime.GOOS == "linux" {
		output, err := utils.ExecuteCommandWithOutput("id", []string{"-u"})
		if err == nil {
			uid, err := strconv.Atoi(strings.TrimSpace(output))
			if err == nil {
				return uid
			}
		}
	}
	return 1000 // Default UID
}

// wordValidator validates input is a single word
func wordValidator(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("input must be a string")
	}

	if len(str) == 0 || strings.Fields(str)[0] != str {
		return fmt.Errorf("%s is not a valid value. It should be a single word", color.YellowString(str))
	}

	return nil
}

// portValidator validates input is a valid port number
func portValidator(val interface{}) error {
	// Handle string input from survey
	if str, ok := val.(string); ok {
		port, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("%s is not a valid port", color.YellowString(str))
		}
		val = port
	}

	// Handle int input
	port, ok := val.(int)
	if !ok {
		return fmt.Errorf("port must be a number")
	}

	if port < 0 || port > 65535 {
		return fmt.Errorf("%d is not a valid port. It should be a number between 0-65535", port)
	}

	return nil
}

// fetchFrameworkVersionTags fetches available framework versions
func fetchFrameworkVersionTags() ([]string, error) {
	versions := []string{"latest"}

	url := os.Getenv("DOCKER_REGISTRY_TAGS_VERSION_URL")
	if url == "" {
		url = "https://registry.hub.docker.com/v2/namespaces/godspeedsystems/repositories/gs-node-service/tags?n=5"
	}

	resp, err := http.Get(url)
	if err != nil {
		color.Red("Not able to connect docker registry. Please check your internet connection.")
		return versions, nil
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return versions, nil
	}

	for _, v := range result.Results {
		versions = append(versions, v.Name)
	}

	return versions, nil
}

// generateProjectFromDotGodspeed generates project files from configuration
func generateProjectFromDotGodspeed(projectName, projectDirPath string, godspeedOptions *GodspeedOptions, exampleName string) error {
	color.Yellow("Generating project files.")

	// Write .godspeed file
	godspeedData, err := json.MarshalIndent(godspeedOptions, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(projectDirPath, ".godspeed"), godspeedData, 0644); err != nil {
		return err
	}

	// Copy dot config files
	if err := utils.CopyDir(filepath.Join(projectDirPath, ".template", "dot-configs"), projectDirPath); err != nil {
		return err
	}

	// Generate package.json, tsconfig.json
	for _, file := range []string{"package.json", "tsconfig.json"} {
		data, err := os.ReadFile(filepath.Join(projectDirPath, ".template", file))
		if err != nil {
			return err
		}

		var packageJSON map[string]interface{}
		if err := json.Unmarshal(data, &packageJSON); err != nil {
			return err
		}

		packageJSON["name"] = projectName

		updatedData, err := json.MarshalIndent(packageJSON, "", "\t")
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(projectDirPath, file), updatedData, 0644); err != nil {
			return err
		}
	}

	// Generate .swcrc file
	swcrcData, err := os.ReadFile(filepath.Join(projectDirPath, ".template", "dot-configs", ".swcrc"))
	if err != nil {
		return err
	}

	var swcrc map[string]interface{}
	if err := json.Unmarshal(swcrcData, &swcrc); err != nil {
		return err
	}

	updatedSwcrc, err := json.MarshalIndent(swcrc, "", "\t")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(projectDirPath, ".swcrc"), updatedSwcrc, 0644); err != nil {
		return err
	}

	// Create folder structure if no example specified
	if exampleName == "" {
		if err := utils.CopyDir(filepath.Join(projectDirPath, ".template", "defaults"), projectDirPath); err != nil {
			return err
		}
	}

	// Compile and copy .devcontainer files
	if err := compileAndCopyDevcontainer(projectDirPath, godspeedOptions); err != nil {
		return err
	}

	color.Green("Successfully generated godspeed project files.\n")
	return nil
}

// compileAndCopyDevcontainer compiles and copies .devcontainer templates
func compileAndCopyDevcontainer(projectDirPath string, godspeedOptions *GodspeedOptions) error {
	// Debug info
	color.Yellow("Preparing to compile and copy .devcontainer files")

	// Create .devcontainer directory
	devcontainerPath := filepath.Join(projectDirPath, ".devcontainer")
	if err := utils.CreateDir(devcontainerPath); err != nil {
		return err
	}

	// Check template directory
	templatePath := filepath.Join(projectDirPath, ".template", ".devcontainer")
	color.Yellow("Checking for template directory: %s", templatePath)

	if !utils.DirExists(templatePath) {
		color.Red("Template directory not found: %s", templatePath)
		color.Yellow("Listing contents of .template directory for debugging:")

		// List .template directory to see what's actually there
		templateDir := filepath.Join(projectDirPath, ".template")
		if utils.DirExists(templateDir) {
			files, err := os.ReadDir(templateDir)
			if err != nil {
				color.Red("Error reading template directory: %v", err)
			} else {
				for _, file := range files {
					color.Yellow("- %s (isDir: %t)", file.Name(), file.IsDir())
				}
			}
		} else {
			color.Red(".template directory not found!")
		}

		// Instead of failing, let's just skip this step for now
		color.Yellow("Skipping .devcontainer setup due to missing templates")
		return nil
	}
	files, err := os.ReadDir(templatePath)
	if err != nil {
		return err
	}

	// Process each template file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		sourcePath := filepath.Join(templatePath, file.Name())
		destPath := filepath.Join(devcontainerPath, file.Name())

		// Check if it's an EJS template
		if strings.HasSuffix(file.Name(), ".ejs") {
			// Read template content
			templateContent, err := os.ReadFile(sourcePath)
			if err != nil {
				return err
			}

			// Process template (simplified version - would need proper EJS library)
			processed := processTemplate(string(templateContent), godspeedOptions)

			// Write processed content to destination (without .ejs extension)
			destPath = strings.TrimSuffix(destPath, ".ejs")
			if err := os.WriteFile(destPath, []byte(processed), 0644); err != nil {
				return err
			}
		} else {
			// Just copy the file
			if err := utils.CopyFile(sourcePath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// processTemplate is a simplified template processor
// In a real implementation, you would use a proper EJS library or Go's template package
func processTemplate(templateContent string, data *GodspeedOptions) string {
	// This is a very simplified implementation
	// You would need a proper template engine in production

	// Convert data to a map for easier access
	dataMap := map[string]interface{}{
		"dockerRegistry":    os.Getenv("DOCKER_REGISTRY"),
		"dockerPackageName": os.Getenv("DOCKER_PACKAGE_NAME"),
		"tag":               data.GSNodeServiceVersion,
		"projectName":       data.ProjectName,
		"servicePort":       data.ServicePort,
		"userUID":           data.UserUID,
		"mongodb":           data.MongoDB,
		"postgresql":        data.PostgreSQL,
		"mysql":             data.MySQL,
		"kafka":             data.Kafka,
		"redis":             data.Redis,
		"elasticsearch":     data.Elasticsearch,
	}

	// Replace placeholders
	result := templateContent
	for key, value := range dataMap {
		placeholder := fmt.Sprintf("<%%= %s %%>", key)

		// Convert value to string based on type
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case int:
			strValue = fmt.Sprintf("%d", v)
		case bool:
			strValue = fmt.Sprintf("%t", v)
		case map[string]interface{}:
			jsonBytes, _ := json.Marshal(v)
			strValue = string(jsonBytes)
		case nil:
			strValue = "false"
		default:
			jsonBytes, _ := json.Marshal(v)
			strValue = string(jsonBytes)
		}

		result = strings.ReplaceAll(result, placeholder, strValue)
	}

	// Remove empty lines
	lines := strings.Split(result, "\n")
	var filteredLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// installDependencies installs project dependencies using npm
func installDependencies(projectDirPath, _ string) error {
	spinner := utils.NewSpinner("Installing dependencies... ")
	spinner.Start()

	cmd := exec.Command("npm", "install", "--quiet", "--no-warnings", "--silent", "--progress=false")
	cmd.Dir = projectDirPath
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	err := cmd.Run()
	spinner.Stop()

	if err != nil {
		return err
	}

	fmt.Println("\nDependencies installed successfully!")
	return nil
}
