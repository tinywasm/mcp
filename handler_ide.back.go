//go:build ignore

package mcp

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/tinywasm/fmt"
)

type IDEInfo struct {
	ID             string
	Name           string
	GetConfigDir   func() (string, error)
	ConfigFileName string
	ServersKey      string
	URLKey          string
	HasInputs       bool
	SkipProfiles    bool
	SupportsHeaders bool
}

func (s *Server) ConfigureIDEs() error {
	return nil
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
			return "", fmt.Err("mcp", "APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Code", "User"), nil
	default:
		return "", fmt.Err("mcp", "unsupported platform: "+runtime.GOOS)
	}
}

func findMCPConfigPaths(basePath string, configFileName string) ([]string, error) {
	return nil, nil
}

func (s *Server) writeMCPConfig(configPath string, ide IDEInfo) (bool, error) {
	return false, nil
}
