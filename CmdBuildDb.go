package wpfinger

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"path"
)

type CmdBuildDb struct {
}

func (builder *CmdBuildDb) Run() error {
	configDir := EnsureAndGetConfigDirectory()
	db, err := gorm.Open(sqlite.Open(path.Join(configDir, "database.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(ComponentFingerprint{}, Vulnerability{})
	if err != nil {
		panic(err)
	}
	log.Println("Updating vulnerabilities from Wordfence ...")
	UpdateWordfenceDB(db)
	log.Println("Vulnerabilities saved in database")
	log.Println("Fingerprinting modules from Wordpress...")
	pluginCount := 0
	for plugin := range Search("popular") {
		log.Printf("Upading %s, current version %s (%d versions found)", plugin.Slug, plugin.Version, len(plugin.Versions))
		for pluginVersion, _ := range plugin.Versions {
			if pluginVersion == "trunk" {
				continue
			}
			var dbPluginCheck ComponentFingerprint
			tx := db.First(&dbPluginCheck, ComponentFingerprint{Slug: plugin.Slug, Version: pluginVersion})
			if tx.Error == nil {
				continue
			}
			if tx.Error != gorm.ErrRecordNotFound {
				panic(tx.Error)
			}
			pluginFingerprint, err := GetPluginFingerprints(plugin, pluginVersion)
			if err != nil {
				continue
			}
			tx = db.First(&dbPluginCheck, ComponentFingerprint{Slug: plugin.Slug, Hash: pluginFingerprint.Hash})
			if tx.Error == nil {
				dbPluginCheck.Fuzzy = true
				tx = db.Save(&dbPluginCheck)
				if tx.Error != nil {
					panic(err)
				}
				pluginFingerprint.Fuzzy = true
			}
			log.Println(pluginFingerprint.Slug, pluginFingerprint.Version, pluginFingerprint.Filename, pluginFingerprint.Hash)
			tx = db.Save(&pluginFingerprint)
			if tx.Error != nil {
				panic(err)
			}
		}
		pluginCount++
	}
	log.Printf("Updated %d plugins", pluginCount)
	return nil
}
