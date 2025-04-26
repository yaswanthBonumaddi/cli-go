package graphql

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

// Definition represents a schema definition
type Definition struct {
	// Definitions can be various complex structures
	// This is a simplified representation
	Properties map[string]interface{} `yaml:"properties,omitempty"`
	Type       string                 `yaml:"type,omitempty"`
	Items      map[string]interface{} `yaml:"items,omitempty"`
	Ref        string                 `yaml:"$ref,omitempty"`
}

// EventSchema represents an event schema
type EventSchema struct {
	Summary     string                   `yaml:"summary,omitempty"`
	Description string                   `yaml:"description,omitempty"`
	Body        map[string]interface{}   `yaml:"body,omitempty"`
	Parameters  []map[string]interface{} `yaml:"parameters,omitempty"`
	Params      []map[string]interface{} `yaml:"params,omitempty"`
	Data        map[string]interface{}   `yaml:"data,omitempty"`
	Responses   map[string]interface{}   `yaml:"responses,omitempty"`
}

// LoadYaml loads yaml files from a directory
func LoadYaml(dirPath string, recursive bool) (map[string]EventSchema, error) {
	result := make(map[string]EventSchema)

	files, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
	if err != nil {
		return nil, err
	}

	// Load all yaml files in the directory
	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var schema EventSchema
		if err := yaml.Unmarshal(data, &schema); err != nil {
			return nil, err
		}

		// Use the filename without extension as the key
		basename := filepath.Base(file)
		key := strings.TrimSuffix(basename, filepath.Ext(basename))
		result[key] = schema
	}

	// If recursive, load from subdirectories
	if recursive {
		subdirs, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}

		for _, subdir := range subdirs {
			if subdir.IsDir() {
				subPath := filepath.Join(dirPath, subdir.Name())
				subResults, err := LoadYaml(subPath, recursive)
				if err != nil {
					return nil, err
				}

				// Add subresults to main results
				for k, v := range subResults {
					result[k] = v
				}
			}
		}
	}

	return result, nil
}

// GenerateSchema generates GraphQL schema from events definitions
func GenerateSchema() {
	if !utils.IsGodspeedProject() {
		return
	}

	// Check for GraphQL event sources
	eventsources, err := findGraphQLEventSources()
	if err != nil {
		color.Red("Error finding GraphQL event sources: %v", err)
		return
	}

	if len(eventsources) == 0 {
		color.Red("No GraphQL event sources found.")
		return
	}

	// Prompt user to select event sources
	var selectedSources []string
	prompt := &survey.MultiSelect{
		Message: "Please select the Graphql Event Sources for which you wish to generate the Graphql schema from Godspeed event defs:",
		Options: eventsources,
	}

	if err := survey.AskOne(prompt, &selectedSources); err != nil {
		color.Red("Error: %v", err)
		return
	}

	if len(selectedSources) == 0 {
		color.Red("Please select at least one GraphQL eventsource")
		return
	}

	// Create Swagger schema and then convert to GraphQL
	for _, eventSource := range selectedSources {
		if err := createGraphQLSchema(eventSource); err != nil {
			color.Red("Error creating GraphQL schema for %s: %v", eventSource, err)
		}
	}
}

// findGraphQLEventSources finds all GraphQL event sources in the project
func findGraphQLEventSources() ([]string, error) {
	var sources []string

	// Look for .yaml files in src/eventsources
	eventsourcesPath := filepath.Join("src", "eventsources")
	if !utils.DirExists(eventsourcesPath) {
		return nil, fmt.Errorf("eventsources directory not found")
	}

	files, err := ioutil.ReadDir(eventsourcesPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		// Extract name without extension
		name := strings.TrimSuffix(file.Name(), ".yaml")

		// Read the file to determine if it's a GraphQL event source
		filePath := filepath.Join(eventsourcesPath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Simple heuristic - check if the file contains GraphQL-related content
		content := string(data)
		if strings.Contains(content, "graphql") || strings.Contains(content, "apollo") {
			sources = append(sources, name)
		}
	}

	return sources, nil
}

// createGraphQLSchema creates a GraphQL schema for the specified event source
func createGraphQLSchema(eventSourceName string) error {
	// Load event schemas
	eventPath := filepath.Join("src", "events")
	definitionsPath := filepath.Join("src", "definitions")

	allEventsSchema, err := LoadYaml(eventPath, true)
	if err != nil {
		return err
	}

	definitions, err := LoadYaml(definitionsPath, false)
	if err != nil {
		// Definitions might not exist, so we can continue
		definitions = make(map[string]EventSchema)
	}

	// Filter events for this event source
	eventSchemas := make(map[string]EventSchema)
	for key, schema := range allEventsSchema {
		parts := strings.Split(key, ".")
		if len(parts) > 0 {
			eventSourceKey := parts[0]

			if eventSourceKey == eventSourceName {
				eventSchemas[key] = schema
				continue
			}

			// Check for multiple event sources ("http & graphql")
			eventSources := strings.Split(eventSourceKey, "&")
			for _, es := range eventSources {
				if strings.TrimSpace(es) == eventSourceName {
					eventSchemas[key] = schema
					break
				}
			}
		}
	}

	if len(eventSchemas) == 0 {
		return fmt.Errorf("did not find any events for the %s eventsource", eventSourceName)
	}

	// Read event source config
	eventsourceConfigPath := filepath.Join("src", "eventsources", fmt.Sprintf("%s.yaml", eventSourceName))
	eventsourceConfigData, err := ioutil.ReadFile(eventsourceConfigPath)
	if err != nil {
		return err
	}

	var eventSourceConfig map[string]interface{}
	if err := yaml.Unmarshal(eventsourceConfigData, &eventSourceConfig); err != nil {
		return err
	}

	// Generate Swagger schema
	swaggerSchema := generateSwaggerJSON(eventSchemas, definitions, eventSourceConfig)

	// Save swagger file to temporary location
	tempDir := os.TempDir()
	swaggerFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-swagger.json", eventSourceName))

	swaggerData, err := json.MarshalIndent(swaggerSchema, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(swaggerFilePath, swaggerData, 0644); err != nil {
		return err
	}

	color.Yellow("Generated and saved swagger schema at temporary location %s. Now generating graphql schema from the same.", swaggerFilePath)

	// Generate GraphQL schema using swagger-to-graphql
	return generateGraphQLSchemaFromSwagger(eventSourceName, swaggerFilePath)
}

// generateSwaggerJSON generates a Swagger JSON schema from event schemas
func generateSwaggerJSON(eventsSchema map[string]EventSchema, definitions map[string]EventSchema, _ map[string]interface{}) map[string]interface{} {
	swaggerCommonPart := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"version":        "0.0.1",
			"title":          "Godspeed: Sample Microservice",
			"description":    "Sample API calls demonstrating the functionality of Godspeed framework",
			"termsOfService": "http://swagger.io/terms/",
			"contact": map[string]interface{}{
				"name":  "Mindgrep Technologies Pvt Ltd",
				"email": "talktous@mindgrep.com",
				"url":   "https://docs.mindgrep.com/docs/microservices/intro",
			},
			"license": map[string]interface{}{
				"name": "Apache 2.0",
				"url":  "https://www.apache.org/licenses/LICENSE-2.0.html",
			},
		},
		"paths": map[string]interface{}{},
	}

	finalSpec := swaggerCommonPart
	paths := finalSpec["paths"].(map[string]interface{})

	// Process each event schema
	for event, schema := range eventsSchema {
		parts := strings.Split(event, ".")
		if len(parts) < 3 {
			continue
		}

		apiEndPoint := parts[2]
		// Convert path parameters from :param format to {param} format
		apiEndPoint = strings.ReplaceAll(apiEndPoint, ":", "{")
		apiEndPoint = strings.ReplaceAll(apiEndPoint, "/{", "/{")
		apiEndPoint = strings.ReplaceAll(apiEndPoint, "}", "}")

		method := parts[1]

		// Initialize method specification
		methodSpec := map[string]interface{}{
			"summary":     schema.Summary,
			"description": schema.Description,
			"requestBody": schema.Body,
			"responses":   schema.Responses,
		}

		// Handle parameters
		if len(schema.Parameters) > 0 {
			methodSpec["parameters"] = schema.Parameters
		} else if len(schema.Params) > 0 {
			methodSpec["parameters"] = schema.Params
		} else if schema.Data != nil && schema.Data["schema"] != nil {
			schemaMap, ok := schema.Data["schema"].(map[string]interface{})
			if ok {
				if params, ok := schemaMap["params"].([]map[string]interface{}); ok {
					methodSpec["parameters"] = params
				}
				if body, ok := schemaMap["body"].(map[string]interface{}); ok {
					methodSpec["requestBody"] = body
				}
			}
		}

		// Set it in the overall schema
		if _, ok := paths[apiEndPoint]; !ok {
			paths[apiEndPoint] = make(map[string]interface{})
		}

		pathMap := paths[apiEndPoint].(map[string]interface{})
		pathMap[method] = methodSpec
	}

	// Add definitions if available
	definitionsMap := make(map[string]interface{})
	for name, def := range definitions {
		// Convert the definition to a map
		defMap := make(map[string]interface{})
		defData, _ := yaml.Marshal(def)
		yaml.Unmarshal(defData, &defMap)
		definitionsMap[name] = defMap
	}

	if len(definitionsMap) > 0 {
		finalSpec["definitions"] = definitionsMap
	}

	return finalSpec
}

// generateGraphQLSchemaFromSwagger generates GraphQL schema from Swagger schema
func generateGraphQLSchemaFromSwagger(eventSourceName, swaggerFilePath string) error {
	outputPath := filepath.Join("src", "eventsources", fmt.Sprintf("%s.graphql", eventSourceName))

	// Use swagger-to-graphql to generate GraphQL schema
	cmd := exec.Command("npx", "swagger-to-graphql", "--swagger-schema="+swaggerFilePath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate GraphQL schema: %v", err)
	}

	// Write GraphQL schema to file
	if err := ioutil.WriteFile(outputPath, output, 0644); err != nil {
		return err
	}

	color.Green("GraphQL schema generated successfully for eventsource %s at %s", eventSourceName, outputPath)
	return nil
}
