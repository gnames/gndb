/*
Copyright © 2025 Dmitry Mozzherin <dmozzherin@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iofs"
	"github.com/gnames/gndb/internal/iologger"
	app "github.com/gnames/gndb/pkg"
	"github.com/gnames/gndb/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	homeDir string
	opts    []config.Option
	cfg     *config.Config
)

// getRootCmd creates and returns the root command.
// Extracted as a function to facilitate testing.
func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Version: fmt.Sprintf("version: %s\nbuild:   %s", app.Version, app.Build),
		Use:     "gndb",
		Short:   "GNdb manages GNverifier database lifecycle",
		Long: `GNdb is a command-line tool for managing the lifecycle of a PostgreSQL
database for GNverifier. It allows users to set up and maintain a local
GNverifier instance with custom data sources.

The tool supports the following functionalities:

- Database Schema Management: Create and migrate the database schema.
- Data Population: Populate the database with nomenclature data.
- Database Optimization: Optimize the database for fast name verification.

Configuration is managed through a config.yaml file, environment variables
(with GNDB_ prefix), and command-line flags.

For more information, see the project's README file.`,
		PersistentPreRunE: bootstrap,
		SilenceErrors:     true,
		SilenceUsage:      true,
	}

	// Remove the automatic "gndb version" prefix
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Override version flag to use -V (consistent with other gn projects)
	rootCmd.Flags().BoolP("version", "V", false, "version for gndb")

	// Add subcommands
	rootCmd.AddCommand(getCreateCmd())

	return rootCmd
}

func bootstrap(cmd *cobra.Command, args []string) error {
	var err error

	homeDir, err = os.UserHomeDir()
	if err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	if err = iofs.EnsureDirs(homeDir); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	// Initialize logging with hardcoded defaults ASAP so all
	// subsequent logs are captured. Will be reconfigured later
	// with user's config settings.
	defaultLog := config.LogConfig{
		Format:      "json",
		Level:       "info",
		Destination: "file",
	}

	if err = iologger.Init(config.LogDir(homeDir), defaultLog, false); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	// Now that logging is initialized, all subsequent logs will be captured
	slog.Info("Bootstrap process started")
	slog.Info("Required directories ensured",
		"config_dir", config.ConfigDir(homeDir),
		"log_dir", config.LogDir(homeDir),
		"cache_dir", config.CacheDir(homeDir))
	slog.Info("Logger initialized with default configuration",
		"format", defaultLog.Format,
		"level", defaultLog.Level,
		"destination", defaultLog.Destination)

	if err = iofs.EnsureConfigFile(homeDir); err != nil {
		slog.Error("Failed to ensure config file", "error", err)
		gn.PrintErrorMessage(err)
		return err
	}

	if err = iofs.EnsureSourcesFile(homeDir); err != nil {
		slog.Error("Failed to ensure sources file", "error", err)
		gn.PrintErrorMessage(err)
		return err
	}

	gn.Info(
		"Configuration files are available at <em>%s</em>",
		config.ConfigDir(homeDir),
	)

	var cfgViper *config.Config
	if cfgViper, err = initConfig(homeDir); err != nil {
		slog.Error("Failed to initialize configuration", "error", err)
		gn.PrintErrorMessage(err)
		return err
	}

	cfg = config.New()
	opts = cfgViper.ToOptions()
	cfg.Update(opts)
	// Set HomeDir after config is loaded
	cfg.Update([]config.Option{config.OptHomeDir(homeDir)})

	// Reconfigure logging with user's settings and proper log file location
	if err = reconfigureLogging(cfg); err != nil {
		slog.Error("Failed to reconfigure logging", "error", err)
		gn.PrintErrorMessage(err)
		return err
	}

	slog.Info("Configuration loaded successfully",
		"config_file", config.ConfigFilePath(homeDir),
		"log_format", cfg.Log.Format,
		"log_level", cfg.Log.Level,
		"log_destination", cfg.Log.Destination,
		"database_host", cfg.Database.Host,
		"database_port", cfg.Database.Port,
		"database_name", cfg.Database.Database,
		"batch_size", cfg.Database.BatchSize,
		"jobs_number", cfg.JobsNumber)

	return nil
}

// reconfigureLogging reinitializes the logger with the loaded configuration.
// Creates log file in the proper location now that we know HomeDir.
// Appends to existing log file to preserve bootstrap logs.
func reconfigureLogging(cfg *config.Config) error {
	logDir := config.LogDir(cfg.HomeDir)
	slog.Info("Reconfiguring logger with user settings",
		"log_dir", logDir,
		"format", cfg.Log.Format,
		"level", cfg.Log.Level,
		"destination", cfg.Log.Destination)

	err := iologger.Init(logDir, cfg.Log, true)
	if err != nil {
		slog.Error("Failed to reconfigure logger", "error", err, "log_dir", logDir)
		return err
	}

	slog.Info("Logger reconfigured successfully", "log_file", logDir+"/gndb.log")
	return nil
}

// Execute adds all child commands to the root command and
// sets flags appropriately. This is called by main.main().
// It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := getRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig(home string) (*config.Config, error) {
	var err error
	cfgPath := config.ConfigFilePath(home)

	v := viper.New()
	v.SetConfigFile(cfgPath)

	initEnvVars(v)

	if err = v.ReadInConfig(); err != nil {
		slog.Error("Failed to read config file", "error", err, "config_path", cfgPath)
		return nil, iofs.ReadFileError(cfgPath, err)
	}

	var res config.Config
	if err = v.Unmarshal(&res); err != nil {
		slog.Error("Failed to unmarshal config", "error", err, "config_path", cfgPath)
		return nil, iofs.ReadFileError(cfgPath, err)
	}
	slog.Info("Configuration unmarshaled successfully",
		"database_host", res.Database.Host,
		"database_port", res.Database.Port,
		"database_name", res.Database.Database,
		"log_level", res.Log.Level,
		"log_format", res.Log.Format,
		"jobs_number", res.JobsNumber)

	return &res, nil
}

func initEnvVars(v *viper.Viper) {
	// Set environment variables we want.
	// We set them manually so we can see clearly which env variables are allowed.
	// These match the fields included in config.ToOptions() - i.e., persistent
	// configuration that can be stored in config.yaml.

	v.SetEnvPrefix("GNDB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Database configuration
	_ = v.BindEnv("database.host", "DATABASE_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT")
	_ = v.BindEnv("database.user", "DATABASE_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD")
	_ = v.BindEnv("database.database", "DATABASE_DATABASE")
	_ = v.BindEnv("database.ssl_mode", "DATABASE_SSL_MODE")
	_ = v.BindEnv("database.batch_size", "DATABASE_BATCH_SIZE")
	slog.Info("Database environment variables bound")

	// Log configuration
	_ = v.BindEnv("log.level", "LOG_LEVEL")
	_ = v.BindEnv("log.format", "LOG_FORMAT")
	_ = v.BindEnv("log.destination", "LOG_DESTINATION")
	slog.Info("Log environment variables bound")

	// General configuration
	_ = v.BindEnv("jobs_number", "JOBS_NUMBER")

	v.AutomaticEnv()
	slog.Info("Environment variable binding complete",
		"automatic_env_lookup", "enabled")
}
