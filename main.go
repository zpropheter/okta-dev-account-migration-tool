package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	
	"github.com/okta/okta-sdk-golang/v5/okta"
	"github.com/spf13/cobra"
)

// Config holds the configuration for connecting to Okta
type Config struct {
	OktaDomain     string
	ConfigFilePath string
	OrgName        string
	Client         *okta.APIClient
}

var (
	configFile  string
	outputDir   string
	inputDir    string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "envsync",
	Short: "A tool for backing up and restoring Okta developer environments",
	Long: `envsync is a tool for backing up and restoring Okta developer environments.
It is only designed and tested for use with Okta developer accounts.`,
}

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup an Okta developer environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig(configFile)
		if err != nil {
			return err
		}
		
		return PerformBackup(cfg, outputDir)
	},
}

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore an Okta developer environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig(configFile)
		if err != nil {
			return err
		}
		
		return PerformRestore(cfg, inputDir)
	},
}

func init() {
	// Add commands to the root command
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	
	// Define flags for the backup command
	backupCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to Okta config file")
	backupCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Directory to store backup files")
	
	// Define flags for the restore command
	restoreCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to Okta config file")
	restoreCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Directory containing backup files")
	restoreCmd.MarkFlagRequired("input")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".okta", "okta.yaml")
}

func LoadConfig(configPath string) (*Config, error) {
	// Create configuration options
	options := []okta.ConfigSetter{}
	
	if configPath != "" {
		fmt.Printf("Using configuration from %s\n", configPath)
	}
	
	sdkConfig, err := okta.NewConfiguration(options...)
	if err != nil {
		return nil, fmt.Errorf("error initializing Okta SDK: %w", err)
	}
	
	client := okta.NewAPIClient(sdkConfig)
	
	domain := extractDomainFromUrl(client.GetConfig().Okta.Client.OrgUrl)
	
	if !IsDeveloperOrg(domain) {
		return nil, fmt.Errorf("this tool is only designed for Okta developer accounts (dev-*.okta.com)")
	}
	
	config := &Config{
		ConfigFilePath: configPath,
		OktaDomain:     domain,
		OrgName:        extractOrgName(domain),
		Client:         client,
	}
	
	ctx := context.Background()
	_, resp, err := client.UserAPI.ListUsers(ctx).Limit(1).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Okta API: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to connect to Okta API: status code %d", resp.StatusCode)
	}
	
	return config, nil
}

func IsDeveloperOrg(domain string) bool {
	devPattern := regexp.MustCompile(`^dev-\d+\.okta\.com$`)
	return devPattern.MatchString(domain)
}

func extractOrgName(domain string) string {
	re := regexp.MustCompile(`^(dev-\d+)\.okta\.com$`)
	matches := re.FindStringSubmatch(domain)
	if len(matches) > 1 {
		return matches[1]
	}
	return domain
}

func extractDomainFromUrl(url string) string {
	return strings.TrimPrefix(url, "https://")
}

func PrepareOktaCliArgs(cfg *Config, args ...string) []string {
    if cfg.ConfigFilePath != "" {
        return append([]string{"--config", cfg.ConfigFilePath}, args...)
    }
    return args
}
