package prisma

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/godspeedsystems/godspeed-cli/internal/utils"
)

// Prepare prepares the Prisma database for use
func Prepare() {
	if !utils.IsGodspeedProject() {
		return
	}

	// Find all Prisma files in the project
	prismaFiles, err := findPrismaFiles()
	if err != nil {
		color.Red("Error finding Prisma files: %v", err)
		return
	}

	if len(prismaFiles) == 0 {
		color.Yellow("No Prisma schema files found.")
		return
	}

	// Generate client and sync database for each Prisma file
	for _, file := range prismaFiles {
		if err := generatePrismaClient(file); err != nil {
			color.Red("Error generating Prisma client for %s: %v", file, err)
			continue
		}

		if err := pushPrismaDb(file); err != nil {
			color.Red("Error pushing Prisma database for %s: %v", file, err)
			continue
		}
	}

	color.Green("Prisma database preparation completed successfully.")
}

// findPrismaFiles finds all Prisma schema files in the project
func findPrismaFiles() ([]string, error) {
	var prismaFiles []string

	// Get the absolute path of datasources directory
	datasourcesDir := filepath.Join(".", "src", "datasources")
	if !utils.DirExists(datasourcesDir) {
		return nil, fmt.Errorf("datasources directory not found")
	}

	// Walk through all subdirectories of datasources
	err := filepath.Walk(datasourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file is a Prisma schema file
		if !info.IsDir() && filepath.Ext(path) == ".prisma" {
			// Get the relative path from current directory
			relPath, err := filepath.Rel(".", path)
			if err != nil {
				return err
			}
			prismaFiles = append(prismaFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return prismaFiles, nil
}

// generatePrismaClient generates the Prisma client for a schema
func generatePrismaClient(schemaPath string) error {
	color.Yellow("Generating Prisma client for %s...", schemaPath)
	return utils.ExecuteCommand("npx", []string{
		"--yes",
		"prisma",
		"generate",
		fmt.Sprintf("--schema=%s", schemaPath),
	})
}

// pushPrismaDb syncs the Prisma schema with the database
func pushPrismaDb(schemaPath string) error {
	color.Yellow("Syncing database with schema %s...", schemaPath)
	return utils.ExecuteCommand("npx", []string{
		"--yes",
		"prisma",
		"db",
		"push",
		fmt.Sprintf("--schema=%s", schemaPath),
	})
}
