package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"veda/internal/editor"
	"veda/internal/log"
)

//go:embed help.txt
var helpText string

func main() {
	// Parse command line flags
	debugFlag := flag.Bool("d", false, "Enable debug logging")
	traceFlag := flag.Bool("t", false, "Enable trace logging")
	helpFlag := flag.Bool("h", false, "Show help message")
	helpLongFlag := flag.Bool("help", false, "Show help message")
	cpuprofile := flag.String("cpuprofile", "", "Write CPU profile to file")

	// Set custom usage function
	flag.Usage = func() {
		fmt.Print(helpText)
	}

	flag.Parse()

	// Handle help flags
	if *helpFlag || *helpLongFlag {
		fmt.Print(helpText)
		os.Exit(0)
	}

	// Set up CPU profiling if requested
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create CPU profile: %v\n", err)
			os.Exit(1)
		}
		
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start CPU profile: %v\n", err)
			f.Close()
			os.Exit(1)
		}
		
		// Ensure profile is written on exit
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
			fmt.Fprintf(os.Stderr, "CPU profile written to %s\n", *cpuprofile)
		}()
		
		fmt.Fprintf(os.Stderr, "CPU profiling enabled, writing to %s\n", *cpuprofile)
	}

	// Set up logging
	log.SetupLogging("veda.log")
	defer log.CleanupLogging()

	// Set logging level based on flags
	if *traceFlag {
		for _, logger := range log.AllLoggers {
			logger.MinLevel = log.TRACE
		}
	} else if *debugFlag {
		for _, logger := range log.AllLoggers {
			logger.MinLevel = log.DEBUG
		}
	}

	ed, err := editor.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create editor: %v", err)
		os.Exit(1)
	}
	defer ed.Close()

	// Check if a file was provided as command-line argument
	args := flag.Args()
	if len(args) > 0 {
		path := args[0]
		ed.OpenPath(path)
	}

	ed.Run()
}
