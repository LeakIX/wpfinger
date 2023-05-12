package wpfinger

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/hashicorp/go-version"
	"io"
	"log"
	"net/http"
)

type ComponentFingerprint struct {
	Slug       string `gorm:"primaryKey;autoIncrement:false"`
	Type       string `gorm:"primarykey;autoIncrement:false"`
	Version    string `gorm:"primarykey;autoIncrement:false"`
	Filename   string
	Hash       string `gorm:"index"`
	Fuzzy      bool
	Downloaded int `gorm:"index"`
}

func GetPluginFingerprints(plugin PluginInfo, pluginVersion string) (*ComponentFingerprint, error) {
	parseVersion, err := version.NewVersion(pluginVersion)
	if err != nil {
		return nil, err
	}
	if len(parseVersion.Prerelease()) > 0 {
		return nil, errors.New("pre release")
	}
	var readmeHash string
	var readmeFile string
	for _, readmeFile = range []string{"readme.txt", "README.md", "README.txt", "readme.md"} {
		readmeHash, err = getPluginReadmeHash(readmeFile, plugin.Slug, pluginVersion)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return &ComponentFingerprint{
		Slug:       plugin.Slug,
		Type:       "plugin",
		Version:    pluginVersion,
		Filename:   readmeFile,
		Hash:       readmeHash,
		Downloaded: plugin.Downloaded,
	}, nil
}

func getPluginReadmeHash(readmeFile, pluginName, pluginVersion string) (string, error) {
	tracUrl := fmt.Sprintf("https://plugins.svn.wordpress.org/%s/tags/%s/%s", pluginName, pluginVersion, readmeFile)
	resp, err := http.DefaultClient.Get(tracUrl)
	if err != nil {
		return "", fmt.Errorf("server error while loading %s", readmeFile)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			log.Fatalln(tracUrl)
		}
		return "", fmt.Errorf("file not found %s : %d on %s", readmeFile, resp.StatusCode, tracUrl)
	}
	hasher := sha256.New()
	_, err = io.Copy(hasher, resp.Body)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
