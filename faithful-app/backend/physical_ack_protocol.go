package main

import (
	"encoding/binary"
	"errors"
)

const physicalAckSize = 32
const physicalAckMagic uint32 = 0x45504131
const physicalAckVersion uint32 = 1
const physicalAckMaxAgeMs uint64 = 250

// A fresh ACK means the C++ callback executed the internal steering function.
// It does not establish real wheel angle, visible animation, or crash safety.
type physicalAckReport struct {
	Received bool
	TrailerIndex uint32
	AgeMs uint64
}

func decodePhysicalAck(raw []byte, nowMs uint64) (physicalAckReport,error) {
	var result physicalAckReport
	if len(raw) < physicalAckSize {return result,errors.New("ACK too short")}
	if binary.LittleEndian.Uint32(raw[0:4]) != physicalAckMagic ||
		binary.LittleEndian.Uint32(raw[4:8]) != physicalAckVersion {
		return result,errors.New("unknown ACK protocol")
	}
	begin := binary.LittleEndian.Uint32(raw[8:12])
	end := binary.LittleEndian.Uint32(raw[12:16])
	tick := binary.LittleEndian.Uint64(raw[16:24])
	trailer := binary.LittleEndian.Uint32(raw[24:28])
	reserved := binary.LittleEndian.Uint32(raw[28:32])
	if begin < 2 || begin%2!=0 || begin!=end || trailer>=20 || reserved!=0 {
		return result,errors.New("torn or invalid ACK")
	}
	if tick>nowMs || nowMs-tick>physicalAckMaxAgeMs {
		return result,errors.New("expired ACK")
	}
	return physicalAckReport{Received:true,TrailerIndex:trailer,AgeMs:nowMs-tick},nil
}
