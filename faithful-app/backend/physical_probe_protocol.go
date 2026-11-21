package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Original.8 R&D: SDK initialization proof only. A recognized DLL is NOT
// evidence that ATS's undocumented steering offsets or hooks are compatible.
const physicalProbeSize = 32
const physicalProbeMagic uint32 = 0x45505038
const physicalProbeVersion uint32 = 8
const physicalProbeMaxAgeMs uint64 = 24 * 60 * 60 * 1000

type physicalProbeReport struct {
	Loaded      bool
	Recognized  bool
	GameVersion string
	Reason      string
}

func decodePhysicalProbe(raw []byte, nowMs uint64) (physicalProbeReport, error) {
	var result physicalProbeReport
	if len(raw) < physicalProbeSize {
		return result, errors.New("DLL probe too short")
	}
	if binary.LittleEndian.Uint32(raw[0:4]) != physicalProbeMagic ||
		binary.LittleEndian.Uint32(raw[4:8]) != physicalProbeVersion {
		return result, errors.New("unknown DLL probe protocol")
	}
	major := binary.LittleEndian.Uint32(raw[8:12])
	minor := binary.LittleEndian.Uint32(raw[12:16])
	accepted := binary.LittleEndian.Uint32(raw[16:20])
	reserved := binary.LittleEndian.Uint32(raw[20:24])
	sinceBoot := binary.LittleEndian.Uint64(raw[24:32])
	if sinceBoot > nowMs || nowMs-sinceBoot > physicalProbeMaxAgeMs {
		return result, errors.New("stale DLL initialization")
	}
	if accepted != 1 || reserved != 0 {
		return result, errors.New("invalid SDK acceptance record")
	}
	result.Loaded = true
	result.GameVersion = fmt.Sprintf("%d.%d", major, minor)
	result.Recognized = major == 1 && minor == 62
	if result.Recognized {
		result.Reason = "DLL chargée, initialisation SDK ATS 1.62 confirmée ; physique non validée"
	} else {
		result.Reason = "DLL détectée, version ATS non prise en charge ; physique interdite"
	}
	return result, nil
}
