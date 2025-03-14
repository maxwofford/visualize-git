package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"runtime/pprof"

	"tree-it/git"
	"tree-it/utils"
)

func main() {
	// Start CPU profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Printf("Could not create CPU profile: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Printf("Could not start CPU profile: %v\n", err)
		os.Exit(1)
	}
	defer pprof.StopCPUProfile()

	startTime := time.Now()
	
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <repo_url>")
		os.Exit(1)
	}

	repoURL := os.Args[1]
	outputFile := fmt.Sprintf("outputs/%s.json", utils.GetRepoNameFromURL(repoURL))

	// Create outputs directory if it doesn't exist
	err = os.MkdirAll("outputs", 0755)
	if err != nil {
		fmt.Printf("Failed to create outputs directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing repository %s...\n", repoURL)
	cloneStart := time.Now()
	
	// Get or clone repository
	repo, err := git.GetOrCloneRepo(repoURL)
	if err != nil {
		fmt.Printf("Repository error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Repository ready in %v\n", time.Since(cloneStart))

	fmt.Println("Processing repository...")
	processStart := time.Now()
	
	data, err := git.ProcessRepo(repo, repoURL)
	if err != nil {
		fmt.Printf("Processing error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Processing completed in %v\n", time.Since(processStart))
	
	fmt.Println("Writing output file...")
	writeStart := time.Now()
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Write completed in %v\n", time.Since(writeStart))
	fmt.Printf("Total time: %v\n", time.Since(startTime))

	// Memory profile at the end
	f2, err := os.Create("mem.prof")
	if err != nil {
		fmt.Printf("Could not create memory profile: %v\n", err)
		os.Exit(1)
	}
	defer f2.Close()
	if err := pprof.WriteHeapProfile(f2); err != nil {
		fmt.Printf("Could not write memory profile: %v\n", err)
		os.Exit(1)
	}
}
