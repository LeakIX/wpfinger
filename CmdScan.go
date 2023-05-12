package wpfinger

import (
	"fmt"
	"github.com/k0kubun/go-ansi"
	"github.com/mitchellh/colorstring"
	"github.com/schollz/progressbar/v3"
	"strings"
)

type CmdScan struct {
	Url string `help:"Url to scan." short:"u" required:""`
	All bool   `short:"a" help:"Scan for all plugins, default is vulnerable only." default:"false"`
	bar *progressbar.ProgressBar
}

func (cmd *CmdScan) Run() error {
	ansiStdout := ansi.NewAnsiStdout()
	scanner, err := NewScanner(!cmd.All)
	if err != nil {
		panic(err)
	}
	statusChan := cmd.setupProgressBar(scanner.PluginCount())
	cmd.bar.RenderBlank()
	coreVersion, err := scanner.GetCoreVersion(cmd.Url)
	if err == nil {
		vulnerabilities := scanner.CheckCoreVuln(coreVersion)
		if len(vulnerabilities) == 0 {
			cmd.bar.Clear()
			colorstring.Fprintf(ansiStdout, "Found WordPress Core version [white]%s[reset]\n", coreVersion)
		} else {
			var vulnIds []string
			for _, vuln := range vulnerabilities {
				var vulnTag string
				switch vuln.Severity {
				case "low":
					vulnTag = "[light_yellow]"
				case "medium":
					vulnTag = "[yellow]"
				case "high":
					vulnTag = "[light_red]"
				case "critical":
					vulnTag = "[red]"
				default:
					vulnTag = "[light_magenta]"
				}
				vulnTag += vuln.Id + "[reset]"
				vulnIds = append(vulnIds, vulnTag)
			}
			line := fmt.Sprintf("Found WordPress Core version [white]%s[reset], vulnerable to %s\n", coreVersion, strings.Join(vulnIds, ", "))
			cmd.bar.Clear()
			colorstring.Fprint(ansiStdout, line)
		}
	}
	cmd.bar.Add(1)
	for pluginInfo := range scanner.ScanPlugins(cmd.Url, statusChan) {
		cmd.bar.Clear()
		vulnerabilities := scanner.CheckVulns("plugin", pluginInfo)
		if len(vulnerabilities) == 0 {
			cmd.bar.Clear()
			colorstring.Fprintf(ansiStdout, "Found plugin [white]%s[reset] version [white]%s[reset]\n", pluginInfo.Slug, pluginInfo.Version)
		} else {
			var vulnIds []string
			for _, vuln := range vulnerabilities {
				var vulnTag string
				switch vuln.Severity {
				case "low":
					vulnTag = "[light_yellow]"
				case "medium":
					vulnTag = "[yellow]"
				case "high":
					vulnTag = "[light_red]"
				case "critical":
					vulnTag = "[red]"
				default:
					vulnTag = "[light_magenta]"
				}
				vulnTag += vuln.Id + "[reset]"
				vulnIds = append(vulnIds, vulnTag)
			}
			line := fmt.Sprintf("Found plugin [white]%s[reset], version [white]%s[reset], vulnerable to %s\n", pluginInfo.Slug, pluginInfo.Version, strings.Join(vulnIds, ", "))
			cmd.bar.Clear()
			colorstring.Fprint(ansiStdout, line)
		}

	}
	close(statusChan)
	cmd.bar.Finish()
	cmd.bar.Exit()
	return nil
}

func (cmd *CmdScan) setupProgressBar(pluginCounts int) chan string {
	statusChan := make(chan string)
	cmd.bar = progressbar.NewOptions(pluginCounts+1,
		progressbar.OptionSetVisibility(barVisibility),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetWriter(ansi.NewAnsiStderr()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowIts(),
		progressbar.OptionSetDescription("[cyan][1/2][reset] Finding Core version"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	go func() {
		for status := range statusChan {
			cmd.bar.Describe("[cyan][2/2][reset] Scanning plugin [white]" + status + "[reset]")
			cmd.bar.Add(1)
		}
	}()
	return statusChan
}
