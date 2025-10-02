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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version: fmt.Sprintf("version: %s\nbuild:   %s", app.Version, app.Build),
	Use:     "gndb",
	Short:   "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRunE: bootstrap,
	RunE:              runRoot,
	SilenceErrors:     true,
	SilenceUsage:      true,
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

	// Initialize logging with hardcoded defaults
	// Will be reconfigured later with user's config settings
	defaultLog := config.LogConfig{
		Format:      "json",
		Level:       "info",
		Destination: "file",
	}
	if err = iologger.Init(config.LogDir(homeDir), defaultLog); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	if err = iofs.EnsureConfigFile(homeDir); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	gn.Info(
		"Configuration files are available at <em>%s</em>",
		config.ConfigDir(homeDir),
	)

	var cfgViper *config.Config
	if cfgViper, err = initConfig(homeDir); err != nil {
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
		gn.PrintErrorMessage(err)
		return err
	}

	slog.Info("Configuration loaded", "config_file", config.ConfigFilePath(homeDir))

	return nil
}

// reconfigureLogging reinitializes the logger with the loaded configuration.
// Creates log file in the proper location now that we know HomeDir.
func reconfigureLogging(cfg *config.Config) error {
	logDir := config.LogDir(cfg.HomeDir)
	return iologger.Init(logDir, cfg.Log)
}

func runRoot(cmd *cobra.Command, args []string) error {

	fmt.Printf("CFG: %#v\n", cfg)
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Remove the automatic "gndb version" prefix
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Override version flag to use -V (consistent with other gn projects)
	rootCmd.Flags().BoolP("version", "V", false, "version for gndb")
}

func initConfig(home string) (*config.Config, error) {
	var err error
	cfgPath := config.ConfigFilePath(home)
	v := viper.New()
	v.SetConfigFile(cfgPath)

	initEnvVars(v)

	if err = v.ReadInConfig(); err != nil {
		return nil, iofs.ReadFileError(cfgPath, err)
	}

	var res config.Config
	if err = v.Unmarshal(&res); err != nil {
		return nil, iofs.ReadFileError(cfgPath, err)
	}

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
	v.BindEnv("database.host", "DATABASE_HOST")
	v.BindEnv("database.port", "DATABASE_PORT")
	v.BindEnv("database.user", "DATABASE_USER")
	v.BindEnv("database.password", "DATABASE_PASSWORD")
	v.BindEnv("database.database", "DATABASE_DATABASE")
	v.BindEnv("database.ssl_mode", "DATABASE_SSL_MODE")
	v.BindEnv("database.batch_size", "DATABASE_BATCH_SIZE")

	// Log configuration
	v.BindEnv("log.level", "LOG_LEVEL")
	v.BindEnv("log.format", "LOG_FORMAT")
	v.BindEnv("log.destination", "LOG_DESTINATION")

	// General configuration
	v.BindEnv("jobs_number", "JOBS_NUMBER")

	v.AutomaticEnv()
}
