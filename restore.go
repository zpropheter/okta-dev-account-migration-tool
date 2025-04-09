package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type IDMapping struct {
	Mappings map[string]map[string]string
	FilePath string
}

func NewIDMapping(restoreDir string) *IDMapping {
	return &IDMapping{
		Mappings: make(map[string]map[string]string),
		//TODO put destination org id into mapping filename
		FilePath: filepath.Join(restoreDir, "id_mapping.json"),
	}
}

func (m *IDMapping) AddMapping(resourceType, oldID, newID string) {
	if _, ok := m.Mappings[resourceType]; !ok {
		m.Mappings[resourceType] = make(map[string]string)
	}
	
	m.Mappings[resourceType][oldID] = newID
	
	m.Save()
}

func (m *IDMapping) GetNewID(resourceType, oldID string) (string, bool) {
	resourceMap, ok := m.Mappings[resourceType]
	if !ok {
		return "", false
	}
	
	newID, ok := resourceMap[oldID]
	return newID, ok
}

func (m *IDMapping) Save() error {
	data, err := json.MarshalIndent(m.Mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling ID mapping: %w", err)
	}
	
	return os.WriteFile(m.FilePath, data, 0644)
}

func (m *IDMapping) Load() error {
	if _, err := os.Stat(m.FilePath); os.IsNotExist(err) {
		m.Mappings = make(map[string]map[string]string)
		return nil
	}
	
	data, err := os.ReadFile(m.FilePath)
	if err != nil {
		return fmt.Errorf("error reading ID mapping file: %w", err)
	}
	
	return json.Unmarshal(data, &m.Mappings)
}

type ResourceRestorer interface {
	Restore(cfg *Config, idMapping *IDMapping, inputDir string) error
}

var customRestorers = map[string]ResourceRestorer{
	"applicationGroups": &ApplicationGroupsRestorer{},
	"user":         &UserGroupsRestorer{},
	"roleAssignment": &RoleAssignmentRestorer{},
}

type UserGroupsRestorer struct{}

func (r *UserGroupsRestorer) Restore(cfg *Config, idMapping *IDMapping, inputDir string) error {
	userGroupsDir := filepath.Join(inputDir, "user", "listGroups")
	
	if _, err := os.Stat(userGroupsDir); os.IsNotExist(err) {
		return nil
	}
	
	userDirs, err := os.ReadDir(userGroupsDir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", userGroupsDir, err)
	}
	
	for _, userDir := range userDirs {
		if userDir.IsDir() {
			oldUserID := userDir.Name()
			
			newUserID, ok := idMapping.GetNewID("user", oldUserID)
			if !ok {
				fmt.Printf("Warning: could not find new ID for user %s, skipping group assignments...\n", oldUserID)
				continue
			}
			
			userPath := filepath.Join(userGroupsDir, oldUserID)
			groupFiles, err := os.ReadDir(userPath)
			if err != nil {
				fmt.Printf("Warning: error reading directory %s: %v\n", userPath, err)
				continue
			}
			
			for _, file := range groupFiles {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
					filePath := filepath.Join(userPath, file.Name())
					data, err := os.ReadFile(filePath)
					if err != nil {
						fmt.Printf("Warning: error reading file %s: %v\n", filePath, err)
						continue
					}
					
					var group map[string]interface{}
					if err := json.Unmarshal(data, &group); err != nil {
						fmt.Printf("Warning: error parsing JSON in %s: %v\n", filePath, err)
						continue
					}
					
					oldGroupID, ok := group["id"].(string)
					if !ok {
						fmt.Printf("Warning: missing id in %s\n", filePath)
						continue
					}
					
					newGroupID, ok := idMapping.GetNewID("group", oldGroupID)
					if !ok {
						fmt.Printf("Warning: could not find new ID for group %s\n", oldGroupID)
						continue
					}
					
					fmt.Printf("Adding user %s to group %s...\n", newUserID, newGroupID)
					
					cmd := exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, "group", "addUserToGroup", 
						"--groupId", newGroupID, "--userId", newUserID)...)
					
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					
					if err := cmd.Run(); err != nil {
						fmt.Printf("Warning: failed to add user %s to group %s: %v\n", 
							newUserID, newGroupID, err)
					}
				}
			}
		}
	}
	
	return nil
}

type RoleAssignmentRestorer struct{}

func (r *RoleAssignmentRestorer) Restore(cfg *Config, idMapping *IDMapping, inputDir string) error {
	roleAssignmentsDir := filepath.Join(inputDir, "roleassignment", "listAssignedRolesForUser")
	
	if _, err := os.Stat(roleAssignmentsDir); os.IsNotExist(err) {
		return nil
	}
	
	userDirs, err := os.ReadDir(roleAssignmentsDir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", roleAssignmentsDir, err)
	}
	
	for _, userDir := range userDirs {
		if userDir.IsDir() {
			oldUserID := userDir.Name()
			
			newUserID, ok := idMapping.GetNewID("user", oldUserID)
			if !ok {
				fmt.Printf("Warning: could not find new ID for user %s, skipping role assignments...\n", oldUserID)
				continue
			}
			
			userPath := filepath.Join(roleAssignmentsDir, oldUserID)
			roleFiles, err := os.ReadDir(userPath)
			if err != nil {
				fmt.Printf("Warning: error reading directory %s: %v\n", userPath, err)
				continue
			}
			
			for _, file := range roleFiles {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
					filePath := filepath.Join(userPath, file.Name())
					
					data, err := os.ReadFile(filePath)
					if err != nil {
						fmt.Printf("Warning: error reading file %s: %v\n", filePath, err)
						continue
					}

					var role map[string]interface{}
					if err := json.Unmarshal(data, &role); err != nil {
						fmt.Printf("Warning: error parsing JSON in %s: %v\n", filePath, err)
						continue
					}
					
					roleType, ok := role["type"].(string)
					if !ok {
						fmt.Printf("Warning: missing role type in %s\n", filePath)
						continue
					}
					
					fmt.Printf("Assigning role %s to user %s...\n", roleType, newUserID)
					
					cmd := exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, "role", "assignRoleToUser", 
						"--userId", newUserID, "--type", roleType)...)
					
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					
					if err := cmd.Run(); err != nil {
						fmt.Printf("Warning: failed to assign role %s to user %s: %v\n", 
							roleType, newUserID, err)
					}
				}
			}
		}
	}
	
	return nil
}

type ApplicationGroupsRestorer struct{}

func (r *ApplicationGroupsRestorer) Restore(cfg *Config, idMapping *IDMapping, inputDir string) error {
	assignmentsDir := filepath.Join(inputDir, "applicationgroups", "listApplicationGroupAssignments")
	
	if _, err := os.Stat(assignmentsDir); os.IsNotExist(err) {
		return nil
	}
	
	files, err := os.ReadDir(assignmentsDir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", assignmentsDir, err)
	}
	
	fmt.Printf("Found %d application group assignments to restore\n", len(files))
	
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(assignmentsDir, file.Name())
			
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Warning: error reading file %s: %v\n", filePath, err)
				continue
			}
			
			var assignment map[string]interface{}
			if err := json.Unmarshal(data, &assignment); err != nil {
				fmt.Printf("Warning: error parsing JSON in %s: %v\n", filePath, err)
				continue
			}
			
			oldAppID, ok := assignment["appId"].(string)
			if !ok {
				fmt.Printf("Warning: missing appId in %s\n", filePath)
				continue
			}
			
			oldGroupID, ok := assignment["id"].(string)
			if !ok {
				fmt.Printf("Warning: missing id in %s\n", filePath)
				continue
			}
			
			newAppID, ok := idMapping.GetNewID("application", oldAppID)
			if !ok {
				fmt.Printf("Warning: could not find new ID for application %s\n", oldAppID)
				continue
			}
			
			newGroupID, ok := idMapping.GetNewID("group", oldGroupID)
			if !ok {
				fmt.Printf("Warning: could not find new ID for group %s\n", oldGroupID)
				continue
			}
			
			fmt.Printf("Assigning group %s to application %s...\n", newGroupID, newAppID)
			
			cmd := exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, "applicationGroups", "assignGroupToApplication", 
				"--appId", newAppID, "--groupId", newGroupID, "--restore-from", filePath)...)
			
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err != nil {
				fmt.Printf("Trying alternative assignment method...\n")
				cmd = exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, "applicationGroups", "assignGroupToApplication", 
					"--appId", newAppID, "--groupId", newGroupID, "--data", "{}")...)
				
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				
				if err := cmd.Run(); err != nil {
					fmt.Printf("Warning: failed to assign group %s to application %s: %v\n", 
						newGroupID, newAppID, err)
				}
			}
		}
	}
	
	return nil
}

func PerformRestore(cfg *Config, inputDir string) error {
	idMapping := NewIDMapping(inputDir)
	
	if err := idMapping.Load(); err != nil {
		fmt.Println("Creating new ID mapping")
	}
	
	backupConfig := GetBackupConfig()
	
	fmt.Println("Restoring singleton resources...")
	if err := restoreSingletonResources(cfg, backupConfig, inputDir); err != nil {
		fmt.Printf("Warning: Error during singleton resources restore: %v\n", err)
	}
	
	fmt.Println("Restoring first pass resources...")
	if err := restoreFirstPassResources(cfg, backupConfig, inputDir, idMapping); err != nil {
		fmt.Printf("Warning: Error during first pass resources restore: %v\n", err)
	}
	
	fmt.Println("Restoring second pass resources...")
	if err := restoreSecondPassResources(cfg, backupConfig, inputDir, idMapping); err != nil {
		fmt.Printf("Warning: Error during second pass resources restore: %v\n", err)
	}
	
	fmt.Println("Handling special resource types...")
	for resourceType, restorer := range customRestorers {
		fmt.Printf("Restoring %s...\n", resourceType)
		if err := restorer.Restore(cfg, idMapping, inputDir); err != nil {
			fmt.Printf("Warning: error restoring %s: %v\n", resourceType, err)
		}
	}
	
	fmt.Println("Restore completed successfully!")
	return nil
}

func restoreSingletonResources(cfg *Config, backupConfig *BackupConfig, inputDir string) error {
	for _, resource := range backupConfig.SingletonResources {
		resourceDir := filepath.Join(inputDir, strings.ToLower(resource.Name), resource.GetCommand)
		
		if _, err := os.Stat(resourceDir); os.IsNotExist(err) {
			fmt.Printf("No backup found for %s/%s, skipping...\n", 
				resource.Name, resource.GetCommand)
			continue
		}
		
		fmt.Printf("Restoring %s using %s command...\n", resource.Name, resource.GetCommand)
		
		files, err := os.ReadDir(resourceDir)
		if err != nil {
			fmt.Printf("Warning: error reading directory %s: %v\n", resourceDir, err)
			continue
		}
		
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				filePath := filepath.Join(resourceDir, file.Name())
				
				cmd := exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, resource.Name, "create", 
					"--restore-from", filePath)...)
				
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				
				if err := cmd.Run(); err != nil {
					fmt.Printf("Warning: Failed to restore %s %s: %v\n", 
						resource.Name, file.Name(), err)
				}
			}
		}
	}
	
	return nil
}

func restoreFirstPassResources(cfg *Config, backupConfig *BackupConfig, inputDir string, idMapping *IDMapping) error {
	for _, resource := range backupConfig.FirstPassResources {
		if resource.ListCommand == "" {
			continue
		}
		
		resourceDir := filepath.Join(inputDir, strings.ToLower(resource.Name), "lists")
		
		if _, err := os.Stat(resourceDir); os.IsNotExist(err) {
			fmt.Printf("No backup found for %s/lists, skipping...\n", resource.Name)
			continue
		}
		
		fmt.Printf("Restoring %s resources...\n", resource.Name)
		
		files, err := os.ReadDir(resourceDir)
		if err != nil {
			fmt.Printf("Warning: error reading directory %s: %v\n", resourceDir, err)
			continue
		}
		
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				filePath := filepath.Join(resourceDir, file.Name())
				
				oldID := strings.TrimSuffix(file.Name(), ".json")
				
				fmt.Printf("Restoring %s from previous ID %s...\n", resource.Name, oldID)
				
				newID, err := restoreResource(cfg, resource.Name, filePath)
				if err != nil {
					fmt.Printf("Warning: error restoring %s from %s: %v\n", 
						resource.Name, oldID, err)
					continue
				}
				
				idMapping.AddMapping(resource.Name, oldID, newID)
				fmt.Printf("Mapped %s old ID %s to new ID %s\n", resource.Name, oldID, newID)
			}
		}
	}
	
	return nil
}

func restoreSecondPassResources(cfg *Config, backupConfig *BackupConfig, inputDir string, idMapping *IDMapping) error {
	for _, resource := range backupConfig.SecondPassResources {
		if _, hasCustomHandler := customRestorers[resource.Name]; hasCustomHandler {
			continue
		}
		
		sourceIDParam := getParameterFlagForResource(resource.SourceIDDir)
		
		resourceDir := filepath.Join(inputDir, strings.ToLower(resource.Name), resource.ListCommand)
		
		if _, err := os.Stat(resourceDir); os.IsNotExist(err) {
			fmt.Printf("No backup found for %s/%s, skipping...\n", 
				resource.Name, resource.ListCommand)
			continue
		}
		
		fmt.Printf("Restoring %s using %s command...\n", resource.Name, resource.ListCommand)
		
		subdirs, err := os.ReadDir(resourceDir)
		if err != nil {
			fmt.Printf("Warning: error reading directory %s: %v\n", resourceDir, err)
			continue
		}
		
		for _, subdir := range subdirs {
			if subdir.IsDir() {
				oldSourceID := subdir.Name()
				
				newSourceID, ok := idMapping.GetNewID(resource.SourceIDDir, oldSourceID)
				if !ok {
					fmt.Printf("Warning: could not find new ID for %s %s, skipping...\n", 
						resource.SourceIDDir, oldSourceID)
					continue
				}
				
				subPath := filepath.Join(resourceDir, oldSourceID)
				files, err := os.ReadDir(subPath)
				if err != nil {
					fmt.Printf("Warning: error reading directory %s: %v\n", subPath, err)
					continue
				}
				
				for _, file := range files {
					if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
						filePath := filepath.Join(subPath, file.Name())
						
						var cmd *exec.Cmd
						
						if isAssignmentResource(resource.Name, resource.ListCommand) {
							cmd = buildAssignmentCommand(cfg, resource.Name, sourceIDParam, newSourceID, filePath)
						} else {
							cmd = exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, resource.Name, "create", 
								fmt.Sprintf("--%s", sourceIDParam), newSourceID, 
								"--restore-from", filePath)...)
						}
						
						if cmd == nil {
							fmt.Printf("Warning: Could not determine appropriate command for %s with %s, skipping...\n",
								resource.Name, resource.ListCommand)
							continue
						}
						
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						
						if err := cmd.Run(); err != nil {
							fmt.Printf("Warning: Failed to restore %s for %s %s: %v\n", 
								resource.Name, resource.SourceIDDir, newSourceID, err)
						}
					}
				}
			}
		}
	}
	
	return nil
}

func isAssignmentResource(resourceName, listCommand string) bool {
	assignmentResources := map[string]map[string]bool{
		"user": {
			"listGroups": true,
		},
		"group": {
			"listUsers": true,
			"listAssignedApplicationsFor": true,
		},
	}
	
	if resourceCommands, ok := assignmentResources[resourceName]; ok {
		if isAssignment, ok := resourceCommands[listCommand]; ok && isAssignment {
			return true
		}
	}
	
	return false
}

func buildAssignmentCommand(cfg *Config, resourceName, sourceIDParam, sourceID, filePath string) *exec.Cmd {
	switch resourceName {
	case "user":
		if sourceIDParam == "userId" {
			return exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, "group", "addUserToGroup", 
				"--userId", sourceID, "--groupId", "TARGET_GROUP_ID", 
				"--assignment-file", filePath)...)
		}
	case "group":
		if sourceIDParam == "groupId" {
			return exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, "user", "addUserToGroup", 
				"--groupId", sourceID, "--userId", "TARGET_USER_ID",
				"--assignment-file", filePath)...)
		}
	}
	return nil
}

func restoreResource(cfg *Config, resourceType string, filePath string) (string, error) {
	cmd := exec.Command("okta-cli-client", PrepareOktaCliArgs(cfg, resourceType, "create", "--restore-from", filePath)...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdout pipe: %w", err)
	}
	
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting command: %w", err)
	}
	
	responseData, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("error reading command output: %w", err)
	}
	
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(responseData, &response); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}
	
	id, ok := response["id"].(string)
	if !ok {
		return "", fmt.Errorf("response does not contain ID field")
	}
	
	return id, nil
}
