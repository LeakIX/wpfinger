package wpfinger

import (
	"encoding/json"
	"github.com/hetiansu5/urlquery"
	"log"
	"net/http"
)

func Search(browse string) chan PluginInfo {
	currentPage := 1
	maxPage := 1
	pluginChan := make(chan PluginInfo)
	go func() {
		defer close(pluginChan)
		for {
			if currentPage > maxPage {
				break
			}
			response, err := searchPage(currentPage)
			if err != nil {
				break
			}
			if currentPage == 1 {
				maxPage = response.Info.Pages
			}
			for _, plugin := range response.Plugins {
				pluginChan <- plugin
			}
			currentPage++
		}
	}()
	return pluginChan
}

func searchPage(page int) (*PluginInfoQueryPluginsResponse, error) {
	queryStruct := PluginInfoQueryPlugins{
		Action: "query_plugins",
		Request: PluginInfoQueryPluginsRequest{
			Browse: "popular",
			Page:   page,
			Fields: map[string]bool{
				"versions": true,
			},
		},
	}

	values, err := urlquery.Marshal(queryStruct)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, "http://api.wordpress.org/plugins/info/1.2/?"+string(values), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Wordpress/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)
	var response PluginInfoQueryPluginsResponse
	err = decoder.Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &response, nil
}

type PluginInfoQueryPlugins struct {
	Action  string                        `query:"action"`
	Request PluginInfoQueryPluginsRequest `query:"request"`
}

type PluginInfoQueryPluginsRequest struct {
	Browse string          `query:"browse,omitempty"`
	Search string          `query:"search,omitempty"`
	Tag    string          `query:"tag,omitempty"`
	Author string          `query:"author,omitempty"`
	Page   int             `query:"page"`
	Fields map[string]bool `query:"fields"`
}

type PluginInfoQueryPluginsResponse struct {
	Info struct {
		Page    int `json:"page"`
		Pages   int `json:"pages"`
		Results int `json:"results"`
	} `json:"info"`
	Plugins []PluginInfo `json:"plugins"`
}

type PluginInfo struct {
	Name       string      `json:"name"`
	Slug       string      `json:"slug"`
	Version    string      `json:"version"`
	Versions   VersionList `json:"versions"`
	Downloaded int         `json:"downloaded"`
}

type VersionList map[string]string

func (i *VersionList) UnmarshalJSON(data []byte) error {
	if string(data) == `[]` {
		return nil
	}
	type tmp VersionList
	return json.Unmarshal(data, (*tmp)(i))
}
