//go:build !windows

package main

func readDevices() []device { return []device{} }
