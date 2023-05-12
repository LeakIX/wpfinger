package wpfinger

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

type CmdUpdateDb struct {
}

func (builder *CmdUpdateDb) Run() error {
	configDir := EnsureAndGetConfigDirectory()
	updateUri := "https://media.leakix.net/wpfinger/update.db"
	resp, err := http.Head(updateUri)
	if err != nil {
		panic(err)
	}
	lastModifiedString := resp.Header.Get("Last-Modified")
	lastModifiedTime, err := time.Parse(time.RFC1123, lastModifiedString)
	if err != nil {
		panic(err)
	}
	localDbFile, err := os.Stat(path.Join(configDir, "database.db"))
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if err == nil && !lastModifiedTime.After(localDbFile.ModTime()) && resp.ContentLength == localDbFile.Size() {
		fmt.Println("Database is up to date")
		return nil
	}
	downloadResp, err := http.Get(updateUri)
	if err != nil {
		panic(err)
	}
	if downloadResp.StatusCode != 200 {
		panic("error downloading update: " + resp.Status)
	}
	localDbForUpdate, err := os.OpenFile(path.Join(configDir, "database.db"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer localDbForUpdate.Close()
	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Updating database ...",
	)
	n, err := io.Copy(io.MultiWriter(localDbForUpdate, bar), downloadResp.Body)
	if err != nil {
		panic(err)
	}
	if n != resp.ContentLength {
		panic("download incomplete")
	}
	return nil
}
