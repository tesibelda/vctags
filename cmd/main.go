// This file contains vctags program main function
//
// Author: Tesifonte Belda
// License: GNU-GPL3 license

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/influxdata/telegraf/plugins/common/shim"
	_ "github.com/tesibelda/vctags/plugins/processors/vctags"
)

var configFile = flag.String("config", "", "path to the config file for this plugin")
var showVersion = flag.Bool("version", false, "show vctags version and exit")

// Version cotains the actual version of vcstat
var Version string = ""

// main creates and runs telegraf's shim processor plugin
func main() {
	var err error

	flag.Parse()
	if *showVersion {
		fmt.Println("vctags", Version)
		os.Exit(0)
	}

	s := shim.New()

	if err = s.LoadConfig(configFile); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading shim configuration: %s\n", err)
		os.Exit(1)
	}

	// run a single plugin until stdin closes or we receive a termination signal
	//if err = s.Run(shim.PollIntervalDisabled); err != nil {
	if err = s.RunProcessor(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR running telegraf shim: %s\n", err)
		os.Exit(2)
	}
}
