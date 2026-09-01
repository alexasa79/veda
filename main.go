package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"
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
	changesFlag := false
	flag.BoolVar(&changesFlag, "c", false, "Open project changes")
	flag.BoolVar(&changesFlag, "changes", false, "Open project changes")
	cpuprofile := flag.String("cpuprofile", "", "Write CPU profile to file")

	// Generate default log path with timestamp
	homeDir, _ := os.UserHomeDir()
	cacheDir := fmt.Sprintf("%s/.cache/veda", homeDir)
	// Ensure cache directory exists early
	os.MkdirAll(cacheDir, 0755)
	timestamp := time.Now().Format("20060102_150405")
	defaultLogPath := fmt.Sprintf("%s/logs/veda_%s.log", cacheDir, timestamp)
	logFile := flag.String("log", defaultLogPath, "Path to log file")

	// Set custom usage function
	flag.Usage = func() {
		fmt.Print(helpText)
	}

	_ = flag.CommandLine.Parse(interspersedFlagArgs(flag.CommandLine, os.Args[1:]))

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
	log.SetupLogging(*logFile)
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
	// Do not defer Close; we need to close (exit alternate screen) before printing

	// Check if a file or directory was provided as a command-line argument
	args := flag.Args()
	if changesFlag {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		ed.OpenProjectChanges(path)
	} else if len(args) > 0 {
		path := args[0]
		ed.OpenPath(path)
	}

	ed.Run()
	// Close the editor (exit alternate screen) before printing any error
	ed.Close()
	// Honor editor-requested exit code (e.g., remote connect failure)
	if code := ed.ExitCode(); code != 0 {
		if msg := ed.ExitMessage(); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(code)
	}
}

type boolFlag interface {
	IsBoolFlag() bool
}

// interspersedFlagArgs moves flags before positional arguments so options can
// appear on either side of the file or directory path.
func interspersedFlagArgs(fs *flag.FlagSet, args []string) []string {
	var options []string
	var positionals []string
	foundTerminator := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if foundTerminator || arg == "-" || !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}
		if arg == "--" {
			foundTerminator = true
			continue
		}

		options = append(options, arg)
		name := strings.TrimPrefix(arg, "-")
		name = strings.TrimPrefix(name, "-")
		name, _, hasValue := strings.Cut(name, "=")
		registered := fs.Lookup(name)
		isBoolean := false
		if registered != nil {
			if boolean, ok := registered.Value.(boolFlag); ok {
				isBoolean = boolean.IsBoolFlag()
			}
		}
		if registered != nil && !hasValue && !isBoolean && i+1 < len(args) {
			i++
			options = append(options, args[i])
		}
	}

	if foundTerminator {
		options = append(options, "--")
	}
	return append(options, positionals...)
}
