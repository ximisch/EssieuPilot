//go:build !windows

package main

func startSteeringRuntime(s *server) func() { return func() {} }
