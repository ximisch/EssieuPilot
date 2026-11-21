//go:build !windows

package main

func statusATSRunning() bool { return false }
func statusSteeringACK() bool { return false }
