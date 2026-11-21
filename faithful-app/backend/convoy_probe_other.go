//go:build !windows
package main
func readConvoyProbe() convoyProbeReport {
    return unknownConvoyProbe("disponible uniquement sur Windows avec ATS")
}
