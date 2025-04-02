package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BackupConfigResource defines a resource that can be backed up
type BackupConfigResource struct {
	// Name is the camelCase representation used in CLI commands
	Name string
	ListCommand string
	GetCommand string
	// Flag to indicate if this resource depends on other resources' IDs
	RequiresIDs bool
	// If this resource requires IDs, this is the directory where we can find those IDs
	SourceIDDir string
	// Flag to indicate if this is a singleton resource (no list capability)
	IsSingleton bool
}

// BackupConfig is the main configuration for backup operations
type BackupConfig struct {
	// First pass resources (list commands that don't require IDs)
	FirstPassResources []BackupConfigResource
	// Second pass resources (commands that require IDs from the first pass)
	SecondPassResources []BackupConfigResource
	SingletonResources []BackupConfigResource
}

// GetBackupConfig returns the hardcoded backup configuration
func GetBackupConfig() *BackupConfig {
	backupconfig := &BackupConfig{
		FirstPassResources: []BackupConfigResource{
			// Users and Groups
			{Name: "user", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "group", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "userType", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Applications
			{Name: "application", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Authorization Servers
			{Name: "authorizationServer", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Identity Providers
			{Name: "identityProvider", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Network & Security
			{Name: "networkZone", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "trustedOrigin", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// API
			{Name: "apiToken", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Customization
			{Name: "customDomain", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "customization", ListCommand: "listBrands", GetCommand: "", RequiresIDs: false},
			
			// Hooks
			{Name: "eventHook", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "inlineHook", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "hookKey", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Policies
			// blocked on https://github.com/okta/okta-cli-client/issues/17
			// {Name: "policy", ListCommand: "listPolicies", GetCommand: "get", RequiresIDs: false},
			
			// Roles
			{Name: "role", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Features
			{Name: "feature", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			
			// Schemas
			//401 on these as admin
			//{Name: "schema", ListCommand: "listLogStreams", GetCommand: "getUser", RequiresIDs: false},
			
			// Additional resources with batch-backup capabilities
			// {Name: "authenticator", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "device", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "emailDomain", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "emailServer", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "logStream", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "profileMapping", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "pushProvider", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "realm", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "resourceSet", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			// {Name: "riskProvider", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
			{Name: "template", ListCommand: "listSmss", GetCommand: "getSms", RequiresIDs: false},
			// {Name: "uISchema", ListCommand: "lists", GetCommand: "get", RequiresIDs: false},
		},
		
		// Second pass resources (those that require IDs from first pass resources)
		SecondPassResources: []BackupConfigResource{
			// Group related resources that require group IDs
			{Name: "group", ListCommand: "listUsers", GetCommand: "", RequiresIDs: true, SourceIDDir: "group"},
			//{Name: "group", ListCommand: "listRules", GetCommand: "", RequiresIDs: true, SourceIDDir: "group"},
			{Name: "group", ListCommand: "listAssignedApplicationsFor", GetCommand: "", RequiresIDs: true, SourceIDDir: "group"},
			//{Name: "groupOwner", ListCommand: "lists", GetCommand: "", RequiresIDs: true, SourceIDDir: "group"},
			
			// User related resources that require user IDs
			{Name: "user", ListCommand: "listAppLinks", GetCommand: "", RequiresIDs: true, SourceIDDir: "user"},
			{Name: "user", ListCommand: "listGroups", GetCommand: "", RequiresIDs: true, SourceIDDir: "user"},
			{Name: "user", ListCommand: "listGrants", GetCommand: "", RequiresIDs: true, SourceIDDir: "user"},
			{Name: "user", ListCommand: "listIdentityProviders", GetCommand: "", RequiresIDs: true, SourceIDDir: "user"},
			{Name: "userFactor", ListCommand: "listFactors", GetCommand: "", RequiresIDs: true, SourceIDDir: "user"},
			{Name: "roleAssignment", ListCommand: "listAssignedRolesForUser", GetCommand: "", RequiresIDs: true, SourceIDDir: "user"},
			
			// Application related resources that require application IDs
			// segfaults, query params? 
			// {Name: "applicationGroups", ListCommand: "listApplicationGroupAssignments", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			// {Name: "applicationUsers", ListCommand: "list", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			// {Name: "applicationTokens", ListCommand: "listOAuth2TokensForApplication", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			// {Name: "applicationCredentials", ListCommand: "listApplicationKeys", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			// {Name: "applicationCredentials", ListCommand: "listCsrsForApplication", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			// {Name: "applicationFeatures", ListCommand: "listFeaturesForApplication", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			// {Name: "applicationGrants", ListCommand: "listScopeConsentGrants", GetCommand: "", RequiresIDs: true, SourceIDDir: "application"},
			
			// Authorization server related resources that require server IDs
			{Name: "authorizationServerClaims", ListCommand: "listOAuth2Claims", GetCommand: "", RequiresIDs: true, SourceIDDir: "authorizationServer"},
			{Name: "authorizationServerScopes", ListCommand: "listOAuth2Scopes", GetCommand: "", RequiresIDs: true, SourceIDDir: "authorizationServer"},
			{Name: "authorizationServerPolicies", ListCommand: "list", GetCommand: "", RequiresIDs: true, SourceIDDir: "authorizationServer"},
			{Name: "authorizationServerClients", ListCommand: "listOAuth2ClientsForAuthorizationServer", GetCommand: "", RequiresIDs: true, SourceIDDir: "authorizationServer"},
			
			// Policy related resources that require policy IDs
			{Name: "policy", ListCommand: "listRules", GetCommand: "", RequiresIDs: true, SourceIDDir: "policy"},
			
			// Other resources with dependencies
			{Name: "authorizationServerRules", ListCommand: "listAuthorizationServerPolicyRules", GetCommand: "", RequiresIDs: true, SourceIDDir: "authorizationServerPolicy"},
			{Name: "identityProvider", ListCommand: "listKeys", GetCommand: "", RequiresIDs: true, SourceIDDir: "identityProvider"},
			{Name: "identityProvider", ListCommand: "listSigningKeys", GetCommand: "", RequiresIDs: true, SourceIDDir: "identityProvider"},
		},
		
		// Singleton resources (accessed via get commands)
		SingletonResources: []BackupConfigResource{
			// Organization settings
			{Name: "orgSetting", GetCommand: "gets", IsSingleton: true},
			{Name: "orgSetting", GetCommand: "getOrgPreferences", IsSingleton: true},
			{Name: "orgSetting", GetCommand: "getOktaCommunicationSettings", IsSingleton: true},
			{Name: "orgSetting", GetCommand: "getOrgOktaSupportSettings", IsSingleton: true},
			{Name: "orgSetting", GetCommand: "getThirdPartyAdminSetting", IsSingleton: true},
			{Name: "orgSetting", GetCommand: "getWellknownOrgMetadata", IsSingleton: true},
			
			// Security settings
			{Name: "attackProtection", GetCommand: "getUserLockoutSettings", IsSingleton: true},
			{Name: "attackProtection", GetCommand: "getAuthenticatorSettings", IsSingleton: true},
			{Name: "threatInsight", GetCommand: "getCurrentConfiguration", IsSingleton: true},
			// {Name: "cAPTCHA", GetCommand: "getOrgCaptchaSettings", IsSingleton: true},
			
			// Rate limit settings
			{Name: "rateLimitSettings", GetCommand: "getPerClient", IsSingleton: true},
			{Name: "rateLimitSettings", GetCommand: "getWarningThreshold", IsSingleton: true},
			{Name: "rateLimitSettings", GetCommand: "getAdminNotifications", IsSingleton: true},
			
			// Customization settings
			//{Name: "customization", GetCommand: "getDefaultSignInPage", IsSingleton: true},
			//{Name: "customization", GetCommand: "getDefaultErrorPage", IsSingleton: true},
			//{Name: "customization", GetCommand: "getSignOutPageSettings", IsSingleton: true},
			//{Name: "customization", GetCommand: "getEmailSettings", IsSingleton: true},
		},
	}
	
	return backupconfig
}

func getParameterFlagForResource(resourceName string) string {
    // Map resource names to their appropriate command-line parameter flags
    paramMap := map[string]string{
        "group":                   "groupId",
        "user":                    "userId",
        "application":             "applicationId",
        "authorizationServer":     "authServerId",
        "authorizationServerPolicies": "authServerId",
        "authorizationServerRules": "policyId",
        "identityProvider":        "idpId",
        "policy":                  "policyId",
        "userFactor":              "userId",
        "roleAssignment":          "userId",
        "applicationGroups":       "appId",
        "applicationUsers":        "appId",
        "applicationTokens":       "appId",
        "applicationCredentials":  "appId",
        "applicationFeatures":     "appId",
        "applicationGrants":       "appId",
        "authorizationServerClaims": "authServerId",
        "authorizationServerScopes": "authServerId",
        "authorizationServerClients": "authServerId",
        "groupOwner":              "groupId",
    }
    
    // Return the appropriate parameter or default to "id" if not found
    if param, ok := paramMap[resourceName]; ok {
        return param
    }
    
    // Default fallback
    return "id"
}

// SaveBackupConfig saves the backup configuration to a file
func SaveBackupConfig(filePath string) error {
	backupconfig := GetBackupConfig()
	
	// Create the directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory for config file: %w", err)
	}
	
	// Marshal the config to JSON
	jsonData, err := json.MarshalIndent(backupconfig, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config to JSON: %w", err)
	}
	
	// Write the JSON to file
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("could not write config to file: %w", err)
	}
	
	return nil
}

// LoadBackupConfig loads the backup configuration from a file
func LoadBackupConfig(filePath string) (*BackupConfig, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}
	
	// Unmarshal the JSON
	var backupconfig BackupConfig
	if err := json.Unmarshal(data, &backupconfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %w", err)
	}
	
	return &backupconfig, nil
}

// DefaultBackupConfigPath returns the default path for the backup config file
func DefaultBackupConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "backup-config.json"
	}
	return filepath.Join(home, ".okta", "backup-config.json")
}