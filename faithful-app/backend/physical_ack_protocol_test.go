package main

import (
	"encoding/binary"
	"testing"
)

func mkAck(seq uint32, index uint32, tick uint64) []byte {
	b:=make([]byte,physicalAckSize)
	binary.LittleEndian.PutUint32(b[0:4],physicalAckMagic)
	binary.LittleEndian.PutUint32(b[4:8],physicalAckVersion)
	binary.LittleEndian.PutUint32(b[8:12],seq)
	binary.LittleEndian.PutUint32(b[12:16],seq)
	binary.LittleEndian.PutUint64(b[16:24],tick)
	binary.LittleEndian.PutUint32(b[24:28],index)
	return b
}

func TestAppliedSteeringAckIsFreshAndValid(t *testing.T) {
	got,err:=decodePhysicalAck(mkAck(2,0,900),1000)
	if err!=nil||!got.Received||got.TrailerIndex!=0||got.AgeMs!=100 {
		t.Fatalf("new in-game steering callback ACK not accepted: %+v %v",got,err)
	}
}

func TestAppliedSteeringAckFailsClosed(t *testing.T) {
	for _,c:=range []struct {name string;data []byte; now uint64}{
		{"too short",[]byte{1},1000},
		{"stale",mkAck(2,0,500),1000},
		{"future",mkAck(2,0,1200),1000},
		{"no sequence",mkAck(0,0,900),1000},
		{"odd sequence",mkAck(3,0,900),1000},
		{"out of range trailer",mkAck(2,20,900),1000},
	}{
		t.Run(c.name,func(t *testing.T){
			got,err:=decodePhysicalAck(c.data,c.now)
			if err==nil||got.Received {t.Fatalf("ACK accepted: %+v %v",got,err)}
		})
	}
	mismatch:=mkAck(2,0,900)
	binary.LittleEndian.PutUint32(mismatch[12:16],4)
	if _,err:=decodePhysicalAck(mismatch,1000);err==nil{t.Fatal("mismatched ACK accepted")}
}
