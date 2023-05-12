package wpfinger

import (
	"github.com/adrg/xdg"
	"os"
	"path/filepath"
)

func EnsureAndGetConfigDirectory() string {
	configDir := filepath.Join(xdg.ConfigHome, "wpfinger")
	configDirHandle, err := os.Stat(configDir)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if err == nil && !configDirHandle.IsDir() {
		panic("config directory is not a directory")
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0750)
		if err != nil {
			panic(err)
		}
	}
	return configDir
}
