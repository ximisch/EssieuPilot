package main

import (
	"encoding/binary"
	"errors"
	"math"
)

// EPS1 is an intentionally small, versioned IPC protocol.
// It commands one whole trailer unit, not each axle independently.
const steeringPacketSize = 36
const steeringMagic uint32 = 0x45505331
const steeringProtocol uint32 = 1

// Sequence values are written as odd while in progress; stable records use even.
type steeringPacket struct {
	Seq             uint32
	Enabled         bool
	UptimeMs        uint64
	TrailerIndex    uint32
	NormalizedAngle float32
}

func encodeSteeringPacket(out []byte, p steeringPacket) error {
	if len(out) < steeringPacketSize {
		return errors.New("buffer too short")
	}
	if p.TrailerIndex >= 20 {
		return errors.New("trailer index out of range")
	}
	if math.IsNaN(float64(p.NormalizedAngle)) || math.IsInf(float64(p.NormalizedAngle), 0) || math.Abs(float64(p.NormalizedAngle)) > 1 {
		return errors.New("invalid steering angle")
	}
	if p.Seq%2 != 0 {
		return errors.New("sequence must be even")
	}
	binary.LittleEndian.PutUint32(out[0:4], steeringMagic)
	binary.LittleEndian.PutUint32(out[4:8], steeringProtocol)
	binary.LittleEndian.PutUint32(out[8:12], p.Seq)
	if p.Enabled {
		binary.LittleEndian.PutUint32(out[12:16], 1)
	} else {
		binary.LittleEndian.PutUint32(out[12:16], 0)
	}
	binary.LittleEndian.PutUint64(out[16:24], p.UptimeMs)
	binary.LittleEndian.PutUint32(out[24:28], p.TrailerIndex)
	binary.LittleEndian.PutUint32(out[28:32], math.Float32bits(p.NormalizedAngle))
	binary.LittleEndian.PutUint32(out[32:36], p.Seq)
	return nil
}

// Validate local copies and use a 250ms heartbeat to fail closed.
func decodeSteeringPacket(b []byte, uptimeMs uint64) (steeringPacket, error) {
	var p steeringPacket
	if len(b) < steeringPacketSize {
		return p, errors.New("short packet")
	}
	if binary.LittleEndian.Uint32(b[:4]) != steeringMagic || binary.LittleEndian.Uint32(b[4:8]) != steeringProtocol {
		return p, errors.New("unknown protocol")
	}
	p.Seq = binary.LittleEndian.Uint32(b[8:12])
	if p.Seq%2 != 0 || p.Seq != binary.LittleEndian.Uint32(b[32:36]) {
		return p, errors.New("torn packet")
	}
	p.Enabled = binary.LittleEndian.Uint32(b[12:16]) == 1
	p.UptimeMs = binary.LittleEndian.Uint64(b[16:24])
	p.TrailerIndex = binary.LittleEndian.Uint32(b[24:28])
	p.NormalizedAngle = math.Float32frombits(binary.LittleEndian.Uint32(b[28:32]))
	if p.TrailerIndex >= 20 || math.IsNaN(float64(p.NormalizedAngle)) || math.IsInf(float64(p.NormalizedAngle), 0) || math.Abs(float64(p.NormalizedAngle)) > 1 {
		return steeringPacket{}, errors.New("invalid payload")
	}
	if p.UptimeMs > uptimeMs || uptimeMs-p.UptimeMs > 250 {
		return steeringPacket{}, errors.New("expired heartbeat")
	}
	return p, nil
}

// Deterministic steering law, rates in degrees per second. The actual in-game
// conversion to normalized angle MUST be calibrated for each compatible trailer.
func advanceTrailerAngle(angle, direction, dt, speedKMH, turnRate, centerRate, maxAngle float64, autoCenter bool) float64 {
	if !isFinite(angle) || !isFinite(dt) || !isFinite(direction) || !isFinite(speedKMH) || !isFinite(turnRate) || !isFinite(centerRate) || !isFinite(maxAngle) {
		return 0
	}
	if maxAngle < 0 || maxAngle > 60 || dt < 0 {
		return 0
	}
	dt = math.Min(dt, 0.1)
	direction = math.Max(-1, math.Min(1, direction))
	angle = math.Max(-maxAngle, math.Min(maxAngle, angle))
	if math.Abs(direction) > 1e-6 {
		angle += math.Copysign(math.Min(math.Abs(turnRate*direction*dt), math.Abs(math.Copysign(maxAngle, direction)-angle)), direction)
	} else if autoCenter && math.Abs(speedKMH) >= 0.2 {
		step := math.Max(0, centerRate) * dt
		if math.Abs(angle) <= step {
			angle = 0
		} else {
			angle -= math.Copysign(step, angle)
		}
	}
	return math.Max(-maxAngle, math.Min(maxAngle, angle))
}
func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
