//go:build !windows

package main

import "errors"

type physicalSteeringBridge struct{}

func newPhysicalSteeringBridge() (*physicalSteeringBridge, error) {
	return nil, errors.New("Windows only")
}
func (b *physicalSteeringBridge) publish(enabled bool, index uint32, angle float32) error {
	return errors.New("Windows only")
}
func (b *physicalSteeringBridge) close() {}
