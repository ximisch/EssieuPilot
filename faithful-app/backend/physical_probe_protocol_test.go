package main

import (
	"encoding/binary"
	"strings"
	"testing"
)

func physicalProbeFixture(major, minor, accepted uint32, at uint64) []byte {
	buf := make([]byte, physicalProbeSize)
	binary.LittleEndian.PutUint32(buf[0:4], physicalProbeMagic)
	binary.LittleEndian.PutUint32(buf[4:8], physicalProbeVersion)
	binary.LittleEndian.PutUint32(buf[8:12], major)
	binary.LittleEndian.PutUint32(buf[12:16], minor)
	binary.LittleEndian.PutUint32(buf[16:20], accepted)
	binary.LittleEndian.PutUint64(buf[24:32], at)
	return buf
}

func TestDLLProbeOnlyConfirmsSDKInitialization(t *testing.T) {
	b := physicalProbeFixture(1, 62, 1, 1000)
	got, err := decodePhysicalProbe(b, 1250)
	if err != nil || !got.Loaded || !got.Recognized || got.GameVersion != "1.62" {
		t.Fatalf("valid SDK probe not recognized: %+v %v", got, err)
	}
	if !strings.Contains(got.Reason, "physique non validée") {
		t.Fatal("SDK probe incorrectly describes physical steering as validated")
	}
}

func TestDLLProbeRejectsInvalidAndStalePackets(t *testing.T) {
	tests := []struct{ name string; payload []byte; now uint64 }{
		{"too short", []byte{1,2,3}, 2000},
		{"no sdk acceptance", physicalProbeFixture(1,62,0,1000), 2000},
		{"future timestamp", physicalProbeFixture(1,62,1,3000), 2000},
		{"stale timestamp", physicalProbeFixture(1,62,1,1000), 1000 + physicalProbeMaxAgeMs + 1},
	}
	corrupted := physicalProbeFixture(1,62,1,1000)
	corrupted[0] ^= 0xff
	tests = append(tests, struct {name string;payload []byte;now uint64}{"bad magic",corrupted,2000})
	for _, tt := range tests {
		t.Run(tt.name,func(t *testing.T){
			got,err:=decodePhysicalProbe(tt.payload,tt.now)
			if err==nil || got.Loaded || got.Recognized {
				t.Fatalf("invalid probe accepted: %+v %v",got,err)
			}
		})
	}
}

func TestDLLProbeVersionRecognitionNotPhysicsValidation(t *testing.T) {
	b:=physicalProbeFixture(1,63,1,1000)
	got,err:=decodePhysicalProbe(b,1100)
	if err!=nil || !got.Loaded || got.Recognized || got.GameVersion!="1.63" {
		t.Fatalf("invalid version gate: %+v %v",got,err)
	}
}
