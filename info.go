package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

	"assetgoblin/config"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// Logo is an ASCII art representation of the AssetGoblin logo,
// displayed when the application is run without arguments.
const Logo = `
                          -  --- --  -                          
                        --- ---- --- ---                        
                       --- ----- ---- ---                       
                      ---  ----- ----  ---                      
                     ----- ----  ----  ----                     
  ----               ----- ----- ----  ----              -----  
  ----------         ----- ----  ----  ----         ----------  
   ------------      ---   ----- ----   ---      ------------   
    --+++----------    +-+ ----- ---- +-+    --+-------+++--    
     ---++  ----------  ++ ----- ---- ++  ----++----  ++---     
      ---+++  --------------       ---------++----- +++---      
       ---+++  -------------------------++++-----  +++----      
       +--++++  -----------+-++++++++-----------+  +++---       
        +--++   -----------------------++-------+  +++--        
         +-++  -------####+----------+####------+  ++--         
           ++  -------####+----------+####-----++  +++          
                -------++--------------++------++               
                ------------------------------++                
                 ---------+##+---++##+-------++                 
                  ---------+########+-------+++                 
                   -----------++++-------++++                   
                     ++++------------++++++-                    
                        ++++++++++++++++-                       
`

// printConfig loads and prints the effective runtime configuration as a table.
func printConfig() {
	var conf config.Config
	if err := conf.Load(); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	presets := make([]string, 0, len(conf.Image.Presets))
	for name, preset := range conf.Image.Presets {
		if preset.Height > 0 {
			presets = append(presets, fmt.Sprintf("%s=%dx%d", name, preset.Width, preset.Height))
		} else {
			presets = append(presets, fmt.Sprintf("%s=%d", name, preset.Width))
		}
	}
	slices.Sort(presets)

	rows := [][]string{
		{"used_config_file", conf.UsedConfigFile},
		{"loaded_from_gob", strconv.FormatBool(conf.LoadedFromGob)},
		{"gob_file", config.GobFilePath()},
		{"port", conf.Port},
		{"public_dir", conf.PublicDir},
		{"python_executable", conf.PythonExecutable},
		{"secret", conf.Secret},
		{"rate_limit.limit", strconv.Itoa(conf.RateLimit.Limit)},
		{"rate_limit.ttl", conf.RateLimit.Ttl.String()},
		{"image.avif_through_vips", strconv.FormatBool(conf.Image.AvifThroughVips)},
		{"image.cache_dir", conf.Image.CacheDir},
		{"image.directory", conf.Image.Directory},
		{"image.path", conf.Image.Path},
		{"image.formats", strings.Join(conf.Image.Formats, ", ")},
		{"image.presets", strings.Join(presets, ", ")},
	}

	output := cliTable([]string{"Setting", "Value"}, rows)
	if _, err := lipgloss.Fprintln(os.Stdout, output); err != nil {
		slog.Error("Failed to render config table", "error", err)
		os.Exit(1)
	}
}

// printUpdateNotice shares the same adaptive layout for both CLI entry points.
func printUpdateNotice(version string) {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	lipgloss.Fprintln(os.Stdout, style.Render("Update available: "+version))
}

var (
	cliHeading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cliSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

// cliTable keeps help and configuration tables visually consistent.
func cliTable(headers []string, rows [][]string) string {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(lipgloss.Color("6"))
			}
			return style
		}).Render()
}

func printBranding() {
	lipgloss.Fprintln(os.Stdout, cliHeading.Render(Logo))
	lipgloss.Fprintln(os.Stdout, cliHeading.Render("AssetGoblin"))
	fmt.Println("Serve static files or dynamically manipulated images with ease")
}

// printUsage uses the parser's flag declarations as its source of truth.
func printUsage(w io.Writer) {
	rows := [][]string{{"-h, -help", "List flags", ""}}
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		name, description := flag.UnquoteUsage(f)
		option := "-" + f.Name
		if name != "" {
			option += " " + name
		}
		rows = append(rows, []string{option, description, f.DefValue})
	})
	lipgloss.Fprintln(w, cliHeading.Render("Usage:"))
	lipgloss.Fprintln(w, cliTable([]string{"Flag", "Description", "Default"}, rows))
}

// printVersionInfo retains the version on its own first line for CLI consumers.
func printVersionInfo(w io.Writer) {
	lipgloss.Fprintln(w, cliHeading.Render(Version))
	lipgloss.Fprintln(w, cliHeading.Render("Build:")+" "+BuildTime+" #"+GitCommit)
}

func printFooter() {
	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Homepage:")+" https://github.com/sbolch/AssetGoblin")
	lipgloss.Fprintln(os.Stdout, cliHeading.Render("Version:"))
	printVersionInfo(os.Stdout)
}

func printSuccess(message string) {
	lipgloss.Fprintln(os.Stdout, cliSuccess.Render(message))
}

// printServerSummary describes configured routes before the listener starts.
func printServerSummary(w io.Writer, cfg *config.Config, addr string, routes, codecs []string) {
	sortedList := func(values []string) string {
		values = slices.Clone(values)
		slices.Sort(values)
		if len(values) == 0 {
			return "None"
		}
		return strings.Join(values, ", ")
	}
	rows := [][]string{
		{"Listen address", addr},
		{"Public directory", cfg.PublicDir},
	}
	if len(cfg.Image.Formats) > 0 {
		rows = append(rows, []string{"Image route", cfg.Image.Path}, []string{"Image directory", cfg.Image.Directory}, []string{"Image cache", cfg.Image.CacheDir})
	} else {
		rows = append(rows, []string{"Image processing", "Disabled"})
	}
	rows = append(rows, []string{"Plugin routes", sortedList(routes)}, []string{"Plugin image codecs", sortedList(codecs)})
	lipgloss.Fprintln(w, cliHeading.Render("Starting AssetGoblin"))
	lipgloss.Fprintln(w, cliTable([]string{"Setting", "Value"}, rows))
}

// printReleaseNotes wraps the original release text without interpreting Markdown.
func printReleaseNotes(w io.Writer, body string, width int) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	lipgloss.Fprintln(w, cliHeading.Render("Release notes:"))
	lipgloss.Fprintln(w, lipgloss.NewStyle().Width(max(1, width)).Render(body))
}
