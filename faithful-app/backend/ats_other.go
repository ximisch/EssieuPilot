//go:build !windows

package main

func scanATS() map[string]any {
	return map[string]any{"found": false, "plugin_found": false, "message": "Recherche des installations ATS disponible sous Windows."}
}
