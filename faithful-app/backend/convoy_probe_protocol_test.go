package main
import (
    "encoding/binary"
    "strings"
    "testing"
)
func makeConvoyProbe(flags uint32,offset int32,tick uint64) []byte {
 b:=make([]byte,convoyProbeSize)
 binary.LittleEndian.PutUint32(b[0:4],convoyProbeMagic)
 binary.LittleEndian.PutUint32(b[4:8],convoyProbeVersion)
 binary.LittleEndian.PutUint32(b[8:12],2)
 binary.LittleEndian.PutUint32(b[12:16],flags)
 binary.LittleEndian.PutUint32(b[16:20],uint32(offset))
 binary.LittleEndian.PutUint64(b[20:28],tick)
 binary.LittleEndian.PutUint32(b[28:32],2)
 return b
}
func TestOriginal21ConvoyProbeIsEvidenceNotSoloProof(t *testing.T) {
 for _,c:=range []struct{flags uint32;offset int32;multi,boolSafety bool}{
  {0,0,false,false},{1,0,false,false},{7,-14,true,true},
 } {
  b:=makeConvoyProbe(c.flags,c.offset,1000)
  got,err:=decodeConvoyProbe(b,1010)
  if err!=nil {t.Fatal(err)}
  if !got.Available || got.MultiplayerEvidence!=c.multi ||
      got.NativeSafetyActive!=c.boolSafety ||got.LastTimeOffset!=c.offset {t.Fatalf("bad result %+v",got)}
  if c.flags==1 && !strings.Contains(got.Status,"absence de preuve de Solo") {t.Fatal("zero offset cannot be called Solo")}
 }
}
func TestOriginal21ProbeFailsClosedOnStaleOrInvalidData(t *testing.T) {
 for name,test:=range map[string]func([]byte){
  "stale":func(b []byte){binary.LittleEndian.PutUint64(b[20:28],1)},
  "torn":func(b []byte){binary.LittleEndian.PutUint32(b[28:32],4)},
  "flags":func(b []byte){binary.LittleEndian.PutUint32(b[12:16],255)},
  "unsafe":func(b []byte){binary.LittleEndian.PutUint32(b[12:16],3)},
 } {
  b:=makeConvoyProbe(1,0,1000)
  test(b)
  _,err:=decodeConvoyProbe(b,10000)
  if name!="stale" { // Newer evidence must not be mistaken for an expired sample.
    binary.LittleEndian.PutUint64(b[20:28],10000)
    _,err=decodeConvoyProbe(b,10010)
  }
  if err==nil {t.Fatal(name+": corrupted report accepted")}
 }
}
