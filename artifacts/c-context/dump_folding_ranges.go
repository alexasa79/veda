package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"veda/internal/proxy"
)

func main() {
	// Create proxy client
	client, err := proxy.NewClient()
	if err != nil {
		log.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Get the file path
	filePath := "file.c"
	if len(os.Args) > 1 {
		filePath = os.Args[1]
	}

	// Get absolute path
	absPath, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	fullPath := absPath + "/" + filePath

	fmt.Printf("Getting folding ranges for: %s\n", fullPath)

	// Get folding ranges
	ranges, err := client.GetFoldingRanges(fullPath)
	if err != nil {
		log.Fatalf("Failed to get folding ranges: %v", err)
	}

	fmt.Printf("Got %d folding ranges\n", len(ranges))

	// Create output file
	output := make(map[string]interface{})
	output["file"] = fullPath
	output["ranges"] = ranges

	// Pretty print for console
	fmt.Println("\nFolding Ranges:")
	fmt.Println("===============")
	for i, r := range ranges {
		kind := "<no kind>"
		if r.Kind != nil {
			kind = *r.Kind
		}
		fmt.Printf("[%d] Lines %d-%d (kind: %s)\n", i, r.StartLine, r.EndLine, kind)
	}

	// Write to JSON file
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	outputFile := "folding_ranges.json"
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("\nOutput written to: %s\n", outputFile)
}
