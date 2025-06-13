package dlt

import (
	"errors"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dlt",
		Short: "DLT CLI",
		Long:  "DLT CLI (stub, to be filled in)",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(NewInit())
	return cmd
}

// createDefaultConfig creates a temporary config file with default values
func createDefaultConfig(projectName string) (string, error) {
	config := map[string]any{}

	if projectName != "" {
		config["project_name"] = projectName
	}

	// Create JSON content
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "dlt-config-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary config file: %w", err)
	}

	// Write config to file
	if _, err := tmpFile.Write(bytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close config file: %w", err)
	}

	return tmpFile.Name(), nil
}


func NewInit() *cobra.Command {
	return &cobra.Command{
		Use:     "init [PROJECT_NAME]",
		Short:   "Initialize a new DLT pipeline project",
		Long:    "Initialize a new DLT pipeline project using the dlt template.",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE:    initDLTProject,
	}
}

func initDLTProject(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Extract project name from arguments if provided
	var projectName string
	if len(args) > 0 {
		projectName = args[0]
		cmdio.LogString(ctx, fmt.Sprintf("Project name (provided via command line): %s", projectName))
	}
	// Create a temporary config file with defaults
	configFile, err := createDefaultConfig(projectName)
	if err != nil {
		return err
	}
	defer os.Remove(configFile)

	r := template.Resolver{
		TemplatePathOrUrl: "dlt",
		ConfigFile:        configFile,
		OutputDir:         ".",
	}

	tmpl, err := r.Resolve(ctx)
	if errors.Is(err, template.ErrCustomSelected) {
		cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
		cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/bundles/templates.html to learn more about custom templates.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to resolve dlt template: %w", err)
	}
	defer tmpl.Reader.Cleanup(ctx)

	err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
	if err != nil {
		return fmt.Errorf("failed to create DLT pipeline project: %w", err)
	}

	return nil
}
