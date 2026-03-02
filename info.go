// Package main provides information about the application.
package main

import (
	"assetgoblin/config"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
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
	if err := conf.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	presets := make([]string, 0, len(conf.Image.Presets))
	for name, width := range conf.Image.Presets {
		presets = append(presets, fmt.Sprintf("%s=%s", name, width))
	}
	sort.Strings(presets)

	rows := [][2]string{
		{"used_config_file", conf.UsedConfigFile},
		{"loaded_from_gob", strconv.FormatBool(conf.LoadedFromGob)},
		{"gob_file", config.GobFilePath()},
		{"port", conf.Port},
		{"public_dir", conf.PublicDir},
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\n", row[0], row[1])
	}
	if err := w.Flush(); err != nil {
		log.Fatalf("Failed to render config table: %v", err)
	}
}
