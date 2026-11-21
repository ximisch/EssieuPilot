package main

import (
    "encoding/binary"
    "errors"
    "fmt"
)

const convoyProbeSize = 32
const convoyProbeMagic uint32 = 0x45504331
const convoyProbeVersion uint32 = 1
const convoyProbeMaxAgeMs uint64 = 5000

// A multiplayer time offset is a partial signal only. A zero offset does NOT
// establish Solo mode. Never use this telemetry to grant physical control.
type convoyProbeReport struct {
    Available bool `json:"available"`
    SDKChannelSeen bool `json:"sdk_channel_seen"`
    MultiplayerEvidence bool `json:"multiplayer_evidence"`
    NativeSafetyActive bool `json:"native_safety_active"`
    LastTimeOffset int32 `json:"last_time_offset"`
    Status string `json:"status"`
}

func decodeConvoyProbe(raw []byte, uptimeMs uint64) (convoyProbeReport,error) {
    var out convoyProbeReport
    if len(raw)<convoyProbeSize {return out,errors.New("rapport incomplet")}
    magic:=binary.LittleEndian.Uint32(raw[0:4])
    version:=binary.LittleEndian.Uint32(raw[4:8])
    if magic!=convoyProbeMagic||version!=convoyProbeVersion {
        return out,errors.New("version de sonde multijoueur non reconnue")
    }
    seq:=binary.LittleEndian.Uint32(raw[8:12])
    end:=binary.LittleEndian.Uint32(raw[28:32])
    if seq==0||seq&1!=0||seq!=end {return out,errors.New("trame multijoueur non stable")}
    flags:=binary.LittleEndian.Uint32(raw[12:16])
    if flags & ^uint32(7)!=0 {return out,errors.New("indicateurs invalides")}
    tick:=binary.LittleEndian.Uint64(raw[20:28])
    if tick>uptimeMs||uptimeMs-tick>convoyProbeMaxAgeMs {return out,errors.New("sonde multijoueur inactive ou perimee")}
    out.Available=true
    out.SDKChannelSeen=flags&1!=0
    out.MultiplayerEvidence=flags&2!=0
    out.NativeSafetyActive=flags&4!=0
    out.LastTimeOffset=int32(binary.LittleEndian.Uint32(raw[16:20]))
    if out.MultiplayerEvidence&&!out.NativeSafetyActive {return convoyProbeReport{},errors.New("securite DLL incoherente")}
    if out.MultiplayerEvidence {
        out.Status="Indice de multijoueur détecté ; la DLL bloque le braquage expérimental jusqu'au redémarrage d'ATS"
    } else if out.SDKChannelSeen {
        out.Status="Canal télémétrique lu, mais absence de preuve de Solo ou de Convoi (offset nul)"
    } else {
        out.Status="Canal non encore lu ; état Solo/Convoi inconnu"
    }
    return out,nil
}

func unknownConvoyProbe(why string) convoyProbeReport {
    return convoyProbeReport{Status:fmt.Sprintf("Convoi indéterminé : %s",why)}
}
