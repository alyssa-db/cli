package dlt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

// InstallDLTSymlink creates a symlink named 'dlt' pointing to the real databricks binary.
func InstallDLTSymlink() error {
	path, err := exec.LookPath("databricks")
	if err != nil {
		return errors.New("databricks CLI not found in PATH")
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	dir := filepath.Dir(path)
	dltPath := filepath.Join(dir, "dlt")

	// Check if DLT already exists
	if fi, err := os.Lstat(dltPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(dltPath)
			if err == nil && target == realPath {
				// DLT is already installed, so we can return success
				cmdio.LogString(context.Background(), "dlt successfully installed")
				return nil
			}
		}
		return fmt.Errorf("cannot create symlink: %q already exists", dltPath)
	} else if !os.IsNotExist(err) {
		// Some other error occurred while checking
		return fmt.Errorf("failed to check if %q exists: %w", dltPath, err)
	}

	if err := os.Symlink(realPath, dltPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	cmdio.LogString(context.Background(), "dlt successfully installed")
	return nil
}

func New() *cobra.Command {
	return &cobra.Command{
		Use:    "install-dlt",
		Short:  "Install DLT",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return InstallDLTSymlink()
		},
	}
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

func NewRoot() *cobra.Command {
	ctx := context.Background()
	cmd := root.New(ctx)

	cmd.Use = "dlt"
	cmd.Short = "DLT CLI"
	cmd.Long = "DLT CLI"
	cmd.Run = func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	}

	cmd.AddCommand(NewInit())

	return cmd
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
