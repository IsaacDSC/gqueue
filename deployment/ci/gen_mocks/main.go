package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileInfo struct {
	Path     string
	Pkg      string
	FileName string
}

func main() {
	skipDirs := map[string]bool{
		"vendor":     true,
		"docs":       true,
		"mocks":      true,
		"tmp":        true,
		".git":       true,
		"specs":      true,
		"deployment": true,
		".vscode":    true,
		".idea":      true,
		"examples":   true,
	}

	timeStart := time.Now()

	// Channel to send found files to the workers
	fileCh := make(chan FileInfo, 100)
	var wgWalk sync.WaitGroup
	var wgMock sync.WaitGroup

	const destDir = "./mocks"
	const numWorkers = 5 // Adjust as needed

	// Start the mock generation workers
	for i := 0; i < numWorkers; i++ {
		wgMock.Add(1)
		go func() {
			defer wgMock.Done()
			for f := range fileCh {
				myPkg := fmt.Sprintf("mock%s", f.Pkg)
				folderDir := fmt.Sprintf("%s/%s", destDir, myPkg)
				completeDest := fmt.Sprintf("%s/%s/mock_%s.go", destDir, myPkg, f.FileName)

				os.Mkdir(folderDir, 0755)

				cmd := exec.Command("go", "run", "go.uber.org/mock/mockgen@latest",
					"-source="+f.Path,
					"-destination="+completeDest,
					"-package="+myPkg,
				)
				if err := cmd.Run(); err != nil {
					fmt.Printf("Error generating mock for %s\n", f.FileName)
				} else {
					fmt.Printf("Mock generated: %s\n", completeDest)
				}
			}
		}()
	}

	// Walk in a goroutine to avoid blocking
	wgWalk.Add(1)
	go func() {
		defer wgWalk.Done()
		defer close(fileCh) // Close the channel when walk is done

		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				if skipDirs[info.Name()] {
					fmt.Printf("Skipping directory: %s\n", info.Name())
					return filepath.SkipDir
				}
				return nil
			}

			if filepath.Ext(path) != ".go" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
				return nil
			}

			if strings.Contains(string(content), "interface {") {
				fileCh <- FileInfo{
					Path:     path,
					Pkg:      extractPackage(string(content)),
					FileName: strings.ReplaceAll(filepath.Base(path), ".go", ""),
				}
			}

			return nil
		})
	}()

	// Wait for the walk to finish (which closes the channel)
	wgWalk.Wait()
	// Wait for all workers to finish
	wgMock.Wait()

	fmt.Printf("\nTotal execution time: %s\n", time.Since(timeStart))
}

func extractPackage(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return strings.TrimPrefix(line, "package ")
		}
	}
	return ""
}
