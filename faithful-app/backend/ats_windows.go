//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var steamPathRx = regexp.MustCompile(`(?i)"path"\s+"([^"]+)"`)

func steamRoots() []string {
	roots := []string{}
	if p := os.Getenv("PROGRAMFILES(X86)"); p != "" {
		roots = append(roots, filepath.Join(p, "Steam"))
	}
	if p := os.Getenv("PROGRAMFILES"); p != "" {
		roots = append(roots, filepath.Join(p, "Steam"))
	}
	for _, drive := range []string{"C:", "D:", "E:", "F:", "G:"} {
		roots = append(roots, drive+`\Steam`, drive+`\Program Files (x86)\Steam`)
	}
	return roots
}

func scanATS() map[string]any {
	seen := map[string]bool{}
	candidates := []string{}
	for _, root := range steamRoots() {
		candidates = append(candidates, filepath.Join(root, "steamapps", "common", "American Truck Simulator"))
		data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
		if err != nil {
			continue
		}
		for _, m := range steamPathRx.FindAllStringSubmatch(string(data), -1) {
			lib := strings.ReplaceAll(m[1], `\\`, `\`)
			candidates = append(candidates, filepath.Join(lib, "steamapps", "common", "American Truck Simulator"))
		}
	}
	for _, game := range candidates {
		game = filepath.Clean(game)
		if seen[strings.ToLower(game)] {
			continue
		}
		seen[strings.ToLower(game)] = true
		stat, err := os.Stat(filepath.Join(game, "bin", "win_x64"))
		if err != nil || !stat.IsDir() {
			continue
		}
		plugins := filepath.Join(game, "bin", "win_x64", "plugins")
		foundPlugin := false
		entries, _ := os.ReadDir(plugins)
		for _, entry := range entries {
			name := strings.ToLower(entry.Name())
			if !entry.IsDir() && strings.HasSuffix(name, ".dll") && (strings.Contains(name, "telemetry") || strings.Contains(name, "scs")) {
				foundPlugin = true
				break
			}
		}
		status := "ATS détecté : " + game + ". "
		if foundPlugin {
			status += "Une DLL de télémétrie possible est présente (fonctionnement à tester)."
		} else {
			status += "Pas de DLL de télémétrie reconnue ; aucun accès direct à la physique."
		}
		return map[string]any{"found": true, "plugin_found": foundPlugin, "message": status, "path": game}
	}
	return map[string]any{"found": false, "plugin_found": false, "message": fmt.Sprintf("ATS non localisé dans %d emplacements Steam examinés. Les bibliothèques non standard peuvent nécessiter une recherche manuelle.", len(seen))}
}
