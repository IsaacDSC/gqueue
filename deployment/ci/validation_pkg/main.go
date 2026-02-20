package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

	folders := make(map[string][]string)
	hasError := false

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}

			if info.Name() != "." {
				folders[info.Name()] = append(folders[info.Name()], path)
			}
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		// Ignora arquivos de teste
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao ler %s: %v\n", path, err)
			return nil
		}

		pkg := extractPackage(string(content))

		// Ignora package main
		if pkg == "main" {
			return nil
		}

		folderName := filepath.Base(filepath.Dir(path))
		if pkg != "" && folderName != "." && pkg != folderName {
			fmt.Printf("ERRO: package '%s' não corresponde ao nome da pasta '%s' em: %s\n", pkg, folderName, path)
			hasError = true
		}

		return nil
	})

	// Valida pastas com o mesmo nome
	for name, paths := range folders {
		if len(paths) > 1 {
			fmt.Printf("ERRO: pasta '%s' está duplicada em:\n", name)
			for _, p := range paths {
				fmt.Printf("  - %s\n", p)
			}
			hasError = true
		}
	}

	if hasError {
		os.Exit(1)
	}

	fmt.Println("Nenhum problema encontrado.")
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
