package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PerformBackup performs the backup operation using the okta-cli-client
func PerformBackup(cfg *Config, outputDir string) error {
	// Set default output directory if not specified
	if outputDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting user home directory: %w", err)
		}
		outputDir = filepath.Join(home, ".okta", cfg.OrgName)
	}
	
	// Get backup config
	backupConfig := GetBackupConfig()
	
	// Process first pass resources (resources that don't require IDs)
	fmt.Println("Backing up first pass resources...")
	for _, resource := range backupConfig.FirstPassResources {
		fmt.Printf("Backing up %s using %s command...\n", resource.Name, resource.ListCommand)
		
		// Construct and execute the command
		cmd := exec.Command("okta-cli-client", resource.Name, resource.ListCommand, 
			"--batch-backup", "--backup-dir", outputDir)
		
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to execute %s %s backup: %v\n", resource.Name, resource.ListCommand, err)
			// Continue with next resource rather than failing the entire backup
			continue
		}
	}
	
	// Process singleton resources (resources that are accessed via get commands)
	fmt.Println("Backing up singleton resources...")
	for _, resource := range backupConfig.SingletonResources {
		fmt.Printf("Backing up %s using %s command...\n", resource.Name, resource.GetCommand)
		
		// For singletons, we call the get command directly
		cmd := exec.Command("okta-cli-client", resource.Name, resource.GetCommand, 
			 "--backup-dir", outputDir)
		
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to execute %s %s backup: %v\n", resource.Name, resource.GetCommand, err)
			// Continue with next resource rather than failing the entire backup
			continue
		}
	}
	
	// Process second pass resources (resources that require IDs from first pass)
	fmt.Println("Backing up second pass resources...")
	if err := backupSecondPassResources(backupConfig, outputDir); err != nil {
		fmt.Printf("Warning: Error during second pass resources backup: %v\n", err)
	}
	
	fmt.Println("Backup completed successfully!")
	return nil
}

// backupSecondPassResources handles the backup of resources that depend on IDs from first pass resources
func backupSecondPassResources(config *BackupConfig, outputDir string) error {
	for _, resource := range config.SecondPassResources {
		// Determine the source directory where we can find the required IDs
		sourceDir := filepath.Join(outputDir, resource.SourceIDDir, "lists")
		
		// Check if the source directory exists
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			fmt.Printf("Warning: Source directory %s not found for %s, skipping...\n", 
				sourceDir, resource.Name)
			continue
		}
		
		// Get IDs from the source directory
		ids, err := getResourceIDsFromDirectory(sourceDir)
		if err != nil {
			fmt.Printf("Warning: Failed to get IDs from %s: %v\n", sourceDir, err)
			continue
		}
		
		if len(ids) == 0 {
			fmt.Printf("No IDs found in %s for %s, skipping...\n", sourceDir, resource.Name)
			continue
		}
		
		fmt.Printf("Found %d IDs for %s in %s\n", len(ids), resource.Name, sourceDir)
		
		// Determine which parameter flag to use based on the resource name
		paramFlag := getParameterFlagForResource(resource.Name)
		
		// Process each ID
		for _, id := range ids {
			fmt.Printf("Backing up %s for %s ID %s using %s command...\n", 
				resource.Name, resource.SourceIDDir, id, resource.ListCommand)
			
			// Construct and execute the command with the appropriate ID parameter
			cmd := exec.Command("okta-cli-client", resource.Name, resource.ListCommand, 
				fmt.Sprintf("--%s", paramFlag), id, "--batch-backup", "--backup-dir", outputDir)
			
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err != nil {
				fmt.Printf("Warning: Failed to execute %s %s backup for ID %s: %v\n", 
					resource.Name, resource.ListCommand, id, err)
				// Continue with next ID rather than failing the entire backup
				continue
			}
		}
	}
	
	return nil
}

// getResourceIDsFromDirectory extracts resource IDs from JSON files in a directory
func getResourceIDsFromDirectory(dirPath string) ([]string, error) {
	var ids []string
	
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".json") {
			// Extract ID from the filename by removing the .json extension
			id := strings.TrimSuffix(entry.Name(), ".json")
			ids = append(ids, id)
		}
	}
	
	return ids, nil
}
