package main

import (
    "strings"
    "testing"
)

func TestOriginalUpdateSelectionUsesOnlyTrustedVersionAndHash(t *testing.T) {
    valid:=originalRelease{
        Tag:"v2.0.0-original.2",Prerelease:true,
        Assets:[]originalAsset{{Name:"Installer_EssieuPilot.exe",
        URL:"https://github.com/ximisch/EssieuPilot/releases/download/v2.0.0-original.2/Installer_EssieuPilot.exe",
        Digest:"sha256:"+strings.Repeat("a",64)}},
    }
    got:=selectOriginalUpdate([]originalRelease{valid},1)
    if got==nil||got.Version!=2{t.Fatal("new trusted original UI build not found")}
    if got=selectOriginalUpdate([]originalRelease{valid},2);got!=nil{t.Fatal("installed build offered again")}
    wrongURL:=valid
    wrongURL.Assets=[]originalAsset{{Name:"Installer_EssieuPilot.exe",
        URL:"https://example.com/fake.exe",Digest:"sha256:"+strings.Repeat("a",64)}}
    if got=selectOriginalUpdate([]originalRelease{wrongURL},1);got!=nil{t.Fatal("untrusted installer URL accepted")}
    wrongDigest:=valid
    wrongDigest.Assets=[]originalAsset{{Name:"Installer_EssieuPilot.exe",
        URL:valid.Assets[0].URL,Digest:"sha256:1234"}}
    if got=selectOriginalUpdate([]originalRelease{wrongDigest},1);got!=nil{t.Fatal("invalid digest accepted")}
    stable:=valid
    stable.Prerelease=false
    if got=selectOriginalUpdate([]originalRelease{stable},1);got!=nil{t.Fatal("stable channel incorrectly mixed with beta")}
    newDesign:=valid
    newDesign.Tag="v2.0.0-native.9"
    if got=selectOriginalUpdate([]originalRelease{newDesign},1);got!=nil{t.Fatal("new UI beta accepted as original UI update")}
}
