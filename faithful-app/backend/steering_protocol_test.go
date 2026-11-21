package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestSteeringPacketABI(t *testing.T) {
	var b [steeringPacketSize]byte
	p := steeringPacket{Seq: 42, Enabled: true, UptimeMs: 1000, TrailerIndex: 2, NormalizedAngle: -0.5}
	if err := encodeSteeringPacket(b[:], p); err != nil {
		t.Fatal(err)
	}
	if steeringPacketSize != 36 || binary.LittleEndian.Uint32(b[:4]) != 0x45505331 || binary.LittleEndian.Uint32(b[24:28]) != 2 {
		t.Fatal("ABI layout")
	}
	got, err := decodeSteeringPacket(b[:], 1210)
	if err != nil || got != p {
		t.Fatalf("roundtrip got %+v err %v", got, err)
	}
	if _, err = decodeSteeringPacket(b[:], 1251); err == nil {
		t.Fatal("expired heartbeat accepted")
	}
	b[8] = 1
	if _, err = decodeSteeringPacket(b[:], 1200); err == nil {
		t.Fatal("torn packet accepted")
	}
}
func TestSteeringPacketValidation(t *testing.T) {
	var b [steeringPacketSize]byte
	if encodeSteeringPacket(b[:], steeringPacket{Seq: 1}) == nil {
		t.Fatal("odd sequence")
	}
	if encodeSteeringPacket(b[:], steeringPacket{Seq: 2, TrailerIndex: 20}) == nil {
		t.Fatal("bad trailer")
	}
	if encodeSteeringPacket(b[:], steeringPacket{Seq: 2, NormalizedAngle: float32(math.NaN())}) == nil {
		t.Fatal("NaN")
	}
}
func TestAdvanceAngleRates(t *testing.T) {
	a := 0.
	for i := 0; i < 100; i++ {
		a = advanceTrailerAngle(a, 1, .05, 0, 14, 7, 35, true)
	}
	if math.Abs(a-35) > 1e-6 {
		t.Fatalf("expected clamp 35, got %v", a)
	}
	a = advanceTrailerAngle(a, 0, .1, 0, 14, 7, 35, true)
	if a != 35 {
		t.Fatalf("stationary shouldn't center: %v", a)
	}
	a = advanceTrailerAngle(a, 0, .1, 2, 14, 7, 35, true)
	if math.Abs(a-34.3) > 1e-5 {
		t.Fatalf("moving center rate incorrect: %v", a)
	}
	a = advanceTrailerAngle(a, -1, .1, 0, 14, 7, 35, true)
	if math.Abs(a-32.9) > 1e-5 {
		t.Fatalf("steering rate incorrect: %v", a)
	}
}
