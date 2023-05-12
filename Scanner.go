package wpfinger

import (
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/glebarez/sqlite"
	"github.com/hashicorp/go-version"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"
)

type Scanner struct {
	db              *gorm.DB
	pluginCache     map[string][]string
	pluginDownloads map[string]int
	sortedPlugins   []string
	httpClient      *http.Client
}

func NewScanner(vulnOnly bool) (*Scanner, error) {
	configDir := EnsureAndGetConfigDirectory()
	db, err := gorm.Open(sqlite.Open(path.Join(configDir, "database.db")), &gorm.Config{})
	if err != nil {
		fmt.Println("Error opening the database. Did you run 'wpfinger update' ?")
		return nil, err
	}
	s := &Scanner{
		db:              db,
		pluginCache:     make(map[string][]string),
		pluginDownloads: make(map[string]int),
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: 10 * time.Second,
		},
	}
	err = s.updatePluginCache(vulnOnly)
	if err != nil {
		return nil, err
	}
	scanPerm := 0
	for plugin, _ := range s.pluginCache {
		err = s.updatePluginCacheFiles(plugin)
		if err != nil {
			return nil, err
		}
		scanPerm += len(s.pluginCache[plugin])
	}
	log.Printf("Loaded %d plugins, %d scan permutations", len(s.pluginCache), scanPerm)
	return s, nil
}

func (s *Scanner) PluginCount() int {
	return len(s.sortedPlugins)
}

func (s *Scanner) GetCoreVersion(rootUrl string) (coreVersion string, err error) {
	rootUrl = strings.TrimRight(rootUrl, "/")
	upgradeUrl := rootUrl + "/wp-admin/upgrade.php"
	req, err := http.NewRequest(http.MethodGet, upgradeUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "WpFinger/0.0.1")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", ErrCoreVersionFailed
	}
	document, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	document.Find("link").Each(func(i int, selection *goquery.Selection) {
		if rel, relExists := selection.Attr("rel"); !relExists || rel != "stylesheet" {
			return
		}
		if href, hrefExists := selection.Attr("href"); !hrefExists {
			return
		} else {
			if !strings.Contains(href, "css/install.min.css?ver=") {
				return
			}
			verParts := strings.Split(href, "?")
			if len(verParts) != 2 {
				return
			}
			ver := strings.TrimPrefix(verParts[1], "ver=")
			parsedVerr, err := version.NewVersion(ver)
			if err != nil {
				return
			}
			coreVersion = parsedVerr.String()
		}
	})
	if len(coreVersion) > 1 {
		return coreVersion, nil
	}
	return "", ErrCoreVersionFailed
}

var ErrCoreVersionFailed = errors.New("core version failed")

func (s *Scanner) ScanPlugins(rootUrl string, status chan string) chan ComponentFingerprint {
	rootUrl = strings.TrimRight(rootUrl, "/")
	pluginInfoChan := make(chan ComponentFingerprint)
	go func() {
		defer close(pluginInfoChan)
		for _, pluginName := range s.sortedPlugins {
			if status != nil {
				status <- pluginName
			}
			files := s.pluginCache[pluginName]
			pluginInfos, err := s.scanPlugin(rootUrl, pluginName, files)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, pluginInfo := range pluginInfos {
				pluginInfoChan <- pluginInfo
			}
		}
	}()
	return pluginInfoChan
}

func (s *Scanner) CheckVulns(componentType string, component ComponentFingerprint) []Vulnerability {
	var vulnsToCheck []Vulnerability
	var vulns []Vulnerability
	tx := s.db.Find(&vulnsToCheck, Vulnerability{
		Slug: component.Slug,
		Type: componentType,
	})
	if tx.Error != nil {
		panic(tx.Error)
	}
	for _, vuln := range vulnsToCheck {
		if vulnerable, err := vuln.Check(component.Version); err == nil && vulnerable {
			vulns = append(vulns, vuln)
		}
	}
	return vulns
}

func (s *Scanner) CheckCoreVuln(coreVersion string) []Vulnerability {
	var vulnsToCheck []Vulnerability
	var vulns []Vulnerability
	tx := s.db.Find(&vulnsToCheck, Vulnerability{
		Type: "core",
	})
	if tx.Error != nil {
		panic(tx.Error)
	}
	for _, vuln := range vulnsToCheck {
		if vulnerable, err := vuln.Check(coreVersion); err == nil && vulnerable {
			vulns = append(vulns, vuln)
		}
	}
	return vulns
}

func (s *Scanner) scanPlugin(rootUrl string, name string, files []string) (pluginFingerprints []ComponentFingerprint, err error) {
	for _, file := range files {
		pluginFileUrl := fmt.Sprintf(rootUrl+"/wp-content/plugins/%s/%s", name, file)
		req, err := http.NewRequest(http.MethodGet, pluginFileUrl, nil)
		req.Header.Set("User-Agent", "WpFinger/0.0.1")
		if err != nil {
			panic(err)
		}
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			continue
		}
		hasher := sha256.New()
		_, err = io.Copy(hasher, resp.Body)
		if err != nil {
			continue
		}
		var pluginInfos []ComponentFingerprint
		tx := s.db.Find(&pluginInfos, ComponentFingerprint{Slug: name, Hash: hex.EncodeToString(hasher.Sum(nil))})
		if tx.Error != nil {
			panic(err)
		}
		for _, pluginInfo := range pluginInfos {
			pluginFingerprints = append(pluginFingerprints, pluginInfo)
		}
	}
	return pluginFingerprints, nil
}

func (s *Scanner) updatePluginCache(vulnOnly bool) (err error) {
	var rows *sql.Rows
	if vulnOnly {
		rows, err = s.db.Table("vulnerabilities").
			Joins("INNER JOIN component_fingerprints ON vulnerabilities.slug = component_fingerprints.slug ").
			Select("component_fingerprints.slug, component_fingerprints.downloaded").
			Where("component_fingerprints.type = ?", "plugin").Group("component_fingerprints.slug").
			Order("component_fingerprints.downloaded DESC").Rows()
	} else {
		rows, err = s.db.Table("component_fingerprints").Select("slug, downloaded").Group("slug").
			Order("downloaded DESC").Where("type = ?", "plugin").Rows()
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var plugin string
		var downloaded int
		err := rows.Scan(&plugin, &downloaded)
		if err != nil {
			return err
		}
		s.pluginCache[plugin] = []string{}
		s.pluginDownloads[plugin] = downloaded
		s.sortedPlugins = append(s.sortedPlugins, plugin)
	}
	sort.Slice(s.sortedPlugins, func(i, j int) bool {
		return s.pluginDownloads[s.sortedPlugins[i]] > s.pluginDownloads[s.sortedPlugins[j]]
	})
	return nil
}

func (s *Scanner) updatePluginCacheFiles(plugin string) (err error) {
	rows, err := s.db.Table("component_fingerprints").Select("filename").Group("filename").Where("slug = ?", plugin).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var file string
		err := rows.Scan(&file)
		if err != nil {
			return err
		}
		s.pluginCache[plugin] = append(s.pluginCache[plugin], file)
	}
	return nil
}
