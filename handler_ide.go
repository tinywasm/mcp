//go:build !wasm

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// IDEInfo represents a supported IDE and its MCP configuration format.
type IDEInfo struct {
	ID             string
	Name           string
	GetConfigDir   func() (string, error)
	ConfigFileName string
	ServersKey     string
	URLKey         string
	ExtraFields    map[string]any
	HasInputs      bool
	SkipProfiles   bool
	SupportsHeaders bool
}

// ConfigureIDEs automatically configures supported IDEs with this MCP server.
func (s *Server) ConfigureIDEs() error {
	ides := []IDEInfo{
		{
			ID:              "vsc",
			Name:            "Visual Studio Code",
			GetConfigDir:    getVSCodeConfigPath,
			ConfigFileName:  "mcp.json",
			ServersKey:      "servers",
			URLKey:          "url",
			ExtraFields:     map[string]any{"type": "http", "autoStart": true},
			HasInputs:       true,
			SupportsHeaders: true,
		},
		{
			ID:              "antigravity",
			Name:            "Antigravity",
			GetConfigDir:    getAntigravityConfigPath,
			ConfigFileName:  "mcp_config.json",
			ServersKey:      "mcpServers",
			URLKey:          "serverUrl",
			ExtraFields:     nil,
			HasInputs:       false,
			SupportsHeaders: true,
		},
		{
			ID:              "claude-code",
			Name:            "Claude Code",
			GetConfigDir:    getClaudeCodeConfigPath,
			ConfigFileName:  ".claude.json",
			ServersKey:      "mcpServers",
			URLKey:          "url",
			ExtraFields:     map[string]any{"type": "http"},
			HasInputs:       false,
			SkipProfiles:    true,
			SupportsHeaders: true,
		},
	}

	updatedIDEs := []string{}

	for _, ide := range ides {
		basePath, err := ide.GetConfigDir()
		if err != nil {
			continue
		}

		var configPaths []string
		if ide.SkipProfiles {
			configPaths = []string{filepath.Join(basePath, ide.ConfigFileName)}
		} else {
			if _, err := os.Stat(basePath); os.IsNotExist(err) {
				if err := os.MkdirAll(basePath, 0755); err != nil {
					continue
				}
			}
			configPaths, err = findMCPConfigPaths(basePath, ide.ConfigFileName)
			if err != nil {
				continue
			}
		}

		ideUpdated := false
		for _, configPath := range configPaths {
			updated, err := s.writeMCPConfig(configPath, ide)
			if err == nil && updated {
				ideUpdated = true
			}
		}
		if ideUpdated {
			updatedIDEs = append(updatedIDEs, ide.Name)
		}
	}

	totalIDEs := len(ides)
	status := fmt.Sprintf("%d of %d IDEs updated", len(updatedIDEs), totalIDEs)
	if len(updatedIDEs) > 0 {
		status = fmt.Sprintf("%s: %s", status, strings.Join(updatedIDEs, ", "))
	}

	s.mu.Lock()
	s.ideStatus = status
	s.mu.Unlock()
	return nil
}

// IDEStatus returns the last ConfigureIDEs result summary.
func (s *Server) IDEStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ideStatus
}

func getVSCodeConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(homeDir, ".config", "Code", "User"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Code", "User"), nil
	default:
		return "", errors.New("unsupported platform: " + runtime.GOOS)
	}
}

func getAntigravityConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gemini", "antigravity"), nil
}

func getClaudeCodeConfigPath() (string, error) {
	return os.UserHomeDir()
}

func findMCPConfigPaths(basePath string, configFileName string) ([]string, error) {
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, errors.New("directory not found")
	}

	profilesPath := filepath.Join(basePath, "profiles")
	if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
		return []string{filepath.Join(basePath, configFileName)}, nil
	}

	entries, err := os.ReadDir(profilesPath)
	if err != nil {
		return nil, err
	}

	configPaths := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			configPaths = append(configPaths, filepath.Join(profilesPath, entry.Name(), configFileName))
		}
	}

	if len(configPaths) == 0 {
		return []string{filepath.Join(basePath, configFileName)}, nil
	}

	return configPaths, nil
}

func needsUpdate(existingEntry map[string]any, newEntry map[string]any, ide IDEInfo) bool {
	existingURL, _ := existingEntry[ide.URLKey].(string)
	newURL, _ := newEntry[ide.URLKey].(string)
	if existingURL != newURL {
		return true
	}
	for k, v := range ide.ExtraFields {
		if !reflect.DeepEqual(existingEntry[k], v) {
			return true
		}
	}
	if !reflect.DeepEqual(existingEntry["headers"], newEntry["headers"]) {
		return true
	}
	return false
}

func (s *Server) writeMCPConfig(configPath string, ide IDEInfo) (bool, error) {
	appName := s.name
	mcpPort := s.port

	if strings.TrimSpace(appName) == "" {
		return false, errors.New("server name cannot be empty")
	}

	var rawConfig map[string]any

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			rawConfig = make(map[string]any)
		} else if os.IsPermission(err) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		if err := json.Unmarshal(data, &rawConfig); err != nil {
			rawConfig = make(map[string]any)
		}
	}

	serversRaw, exists := rawConfig[ide.ServersKey]
	var servers map[string]any
	if exists {
		servers, _ = serversRaw.(map[string]any)
	}
	if servers == nil {
		servers = make(map[string]any)
	}

	expectedURL := fmt.Sprintf("http://localhost:%s/mcp", mcpPort)
	serverID := strings.ToLower(appName)

	duplicatesRemoved := false
	for key, entry := range servers {
		if serverEntry, ok := entry.(map[string]any); ok {
			if url, _ := serverEntry[ide.URLKey].(string); url == expectedURL {
				if key != serverID {
					delete(servers, key)
					duplicatesRemoved = true
				}
			}
		}
	}

	serverEntry := map[string]any{
		ide.URLKey: fmt.Sprintf("http://localhost:%s/mcp", mcpPort),
	}
	for k, v := range ide.ExtraFields {
		serverEntry[k] = v
	}

	s.mu.RLock()
	apiKey := s.apiKey
	s.mu.RUnlock()

	if apiKey != "" && ide.SupportsHeaders {
		serverEntry["headers"] = map[string]string{
			"Authorization": "Bearer " + apiKey,
		}
	}

	if !duplicatesRemoved {
		if existingEntry, hasEntry := servers[serverID]; hasEntry {
			if existing, ok := existingEntry.(map[string]any); ok {
				if !needsUpdate(existing, serverEntry, ide) {
					return false, nil
				}
			}
		}
	}

	servers[serverID] = serverEntry
	rawConfig[ide.ServersKey] = servers

	if ide.HasInputs {
		if _, hasInputs := rawConfig["inputs"]; !hasInputs {
			rawConfig["inputs"] = []any{}
		}
	}

	updatedData, err := json.MarshalIndent(rawConfig, "", "\t")
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		if os.IsPermission(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
