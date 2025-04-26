package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/godspeedsystems/godspeed-cli/internal/create"
	"github.com/godspeedsystems/godspeed-cli/internal/devops"
	"github.com/godspeedsystems/godspeed-cli/internal/graphql"
	"github.com/godspeedsystems/godspeed-cli/internal/otel"
	"github.com/godspeedsystems/godspeed-cli/internal/plugin"
	"github.com/godspeedsystems/godspeed-cli/internal/prisma"
	"github.com/godspeedsystems/godspeed-cli/internal/utils"
	"github.com/spf13/cobra"
)

var version = "1.0.0" // This would be set during build

func main() {
	printBanner()

	rootCmd := &cobra.Command{
		Use:     "godspeed",
		Short:   "Godspeed CLI tool for the Godspeed Framework",
		Version: version,
	}

	// Add create command
	createCmd := &cobra.Command{
		Use:   "create [projectName]",
		Short: "Create a new godspeed project",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fromTemplate, _ := cmd.Flags().GetString("from-template")
			fromExample, _ := cmd.Flags().GetString("from-example")
			create.Execute(args[0], fromTemplate, fromExample, version)
		},
	}
	createCmd.Flags().String("from-template", "", "Create a project from a template")
	createCmd.Flags().String("from-example", "", "Create a project from examples")
	rootCmd.AddCommand(createCmd)

	// Add dev command
	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Run godspeed development server",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.ExecuteCommand("npm", []string{"run", "dev"})
			}
		},
	}
	rootCmd.AddCommand(devCmd)

	// Add clean command
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean the previous build",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.ExecuteCommand("npm", []string{"run", "clean"})
			}
		},
	}
	rootCmd.AddCommand(cleanCmd)

	// Add link command
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Link a local Godspeed project to the global environment for development in godspeed-daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.UpdateServicesJson(true)
			}
		},
	}
	rootCmd.AddCommand(linkCmd)

	// Add unlink command
	unlinkCmd := &cobra.Command{
		Use:   "unlink",
		Short: "Unlink a local Godspeed project from the global environment",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.UpdateServicesJson(false)
			}
		},
	}
	rootCmd.AddCommand(unlinkCmd)

	// Add gen-crud-api command
	genCrudApiCmd := &cobra.Command{
		Use:   "gen-crud-api",
		Short: "Scans your prisma datasources and generate CRUD APIs events and workflows",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.ExecuteCommand("npm", []string{"run", "gen-crud-api"})
			}
		},
	}
	rootCmd.AddCommand(genCrudApiCmd)

	// Add gen-graphql-schema command
	genGraphqlSchemaCmd := &cobra.Command{
		Use:   "gen-graphql-schema",
		Short: "Scans your graphql events and generate graphql schema",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				graphql.GenerateSchema()
			}
		},
	}
	rootCmd.AddCommand(genGraphqlSchemaCmd)

	// Add build command
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build the godspeed project. Create a production build",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.ExecuteCommand("npm", []string{"run", "build"})
			}
		},
	}
	rootCmd.AddCommand(buildCmd)

	// Add preview command
	previewCmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview the production build",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.ExecuteCommand("npm", []string{"run", "preview"})
			}
		},
	}
	rootCmd.AddCommand(previewCmd)

	// Add serve command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Build and preview the production build in watch mode",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				utils.ExecuteCommand("npm", []string{"run", "serve"})
			}
		},
	}
	rootCmd.AddCommand(serveCmd)

	// Add prisma command
	prismaCmd := &cobra.Command{
		Use:   "prisma",
		Short: "Proxy to prisma commands with some add-on commands to handle prisma datasources",
	}
	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare your prisma database for use",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsGodspeedProject() {
				prisma.Prepare()
			}
		},
	}
	prismaCmd.AddCommand(prepareCmd)
	rootCmd.AddCommand(prismaCmd)

	// Add plugin command
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage (add, remove, update) eventsource and datasource plugins for godspeed",
	}

	pluginAddCmd := &cobra.Command{
		Use:   "add [pluginName]",
		Short: "Add an eventsource/datasource plugin",
		Run: func(cmd *cobra.Command, args []string) {
			var pluginName string
			if len(args) > 0 {
				pluginName = args[0]
			}
			plugin.Add(pluginName)
		},
	}

	pluginRemoveCmd := &cobra.Command{
		Use:   "remove [pluginName]",
		Short: "Remove an eventsource/datasource plugin",
		Run: func(cmd *cobra.Command, args []string) {
			var pluginName string
			if len(args) > 0 {
				pluginName = args[0]
			}
			plugin.Remove(pluginName)
		},
	}

	pluginUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an eventsource/datasource plugin",
		Run: func(cmd *cobra.Command, args []string) {
			plugin.Update()
		},
	}

	pluginCmd.AddCommand(pluginAddCmd, pluginRemoveCmd, pluginUpdateCmd)
	rootCmd.AddCommand(pluginCmd)

	// Add devops-plugin command
	devopsPluginCmd := &cobra.Command{
		Use:   "devops-plugin",
		Short: "Manages godspeed devops-plugins",
	}

	devopsPluginInstallCmd := &cobra.Command{
		Use:   "install",
		Short: "Install a godspeed devops plugin",
		Run: func(cmd *cobra.Command, args []string) {
			devops.Install("")
		},
	}

	devopsPluginRemoveCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a godspeed devops plugin",
		Run: func(cmd *cobra.Command, args []string) {
			devops.Remove("")
		},
	}

	devopsPluginListCmd := &cobra.Command{
		Use:   "list",
		Short: "List available godspeed devops plugins",
		Run: func(cmd *cobra.Command, args []string) {
			installed, _ := cmd.Flags().GetBool("installed")
			devops.List(installed)
		},
	}
	devopsPluginListCmd.Flags().Bool("installed", false, "List installed plugins only")

	devopsPluginUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a godspeed devops plugin",
		Run: func(cmd *cobra.Command, args []string) {
			devops.Update()
		},
	}

	devopsPluginCmd.AddCommand(devopsPluginInstallCmd, devopsPluginRemoveCmd, devopsPluginListCmd, devopsPluginUpdateCmd)

	// Add devops plugin subcommands for installed plugins
	home, _ := os.UserHomeDir()
	pluginPath := filepath.Join(home, ".godspeed", "devops-plugins")

	if utils.DirExists(pluginPath) {
		plugins, err := os.ReadDir(pluginPath)
		if err == nil {
			for _, plugin := range plugins {
				if plugin.IsDir() {
					pluginName := plugin.Name()
					pluginCmd := &cobra.Command{
						Use:                pluginName,
						Short:              "Installed godspeed devops plugin",
						DisableFlagParsing: true,
						Run: func(cmd *cobra.Command, args []string) {
							pluginPath := filepath.Join(home, ".godspeed", "devops-plugins", pluginName, "dist", "index.js")
							if utils.FileExists(pluginPath) {
								utils.ExecuteCommand("node", append([]string{pluginPath}, args...))
							} else {
								fmt.Printf("%s is not installed properly. Please make sure %s exists.\n", pluginName, pluginPath)
							}
						},
					}
					devopsPluginCmd.AddCommand(pluginCmd)
				}
			}
		}
	}

	rootCmd.AddCommand(devopsPluginCmd)

	// Add otel command
	otelCmd := &cobra.Command{
		Use:   "otel",
		Short: "Enable/disable Observability in Godspeed",
	}

	otelEnableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable Observability in project",
		Run: func(cmd *cobra.Command, args []string) {
			otel.Enable()
		},
	}

	otelDisableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable Observability in project",
		Run: func(cmd *cobra.Command, args []string) {
			otel.Disable()
		},
	}

	otelCmd.AddCommand(otelEnableCmd, otelDisableCmd)
	rootCmd.AddCommand(otelCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func printBanner() {
	fmt.Println()
	white := color.New(color.FgWhite).SprintFunc()
	boldWhite := color.New(color.FgWhite, color.Bold).SprintFunc()
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	fmt.Printf("%s %s\n", white("       ,_,   "), red("╔════════════════════════════════════╗"))

	fmt.Printf("%s %s\n", boldWhite("      (o o)  "), red("║")+yellow("        Welcome to Godspeed         ")+red("║"))

	fmt.Printf("%s %s\n", blue("     ({___}) "), red("║")+yellow("    World's First Meta Framework    ")+red("║"))
	fmt.Printf("%s %s\n", boldWhite("       \" \"   "), red("╚════════════════════════════════════╝"))
	fmt.Println()
}
