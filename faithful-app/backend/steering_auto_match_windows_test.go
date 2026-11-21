//go:build windows
package main
import ("fmt";"testing")
func TestOriginal14AutoSelectManyExactTrailerProfiles(t *testing.T) {
 p:=runtimeProfile{Trailers:make([]runtimeTrailer,20)}
 for i:=range p.Trailers {
  id:=fmt.Sprintf("ats.vehicle.%02d",i)
  p.Trailers[i]=runtimeTrailer{
    ID:fmt.Sprintf("preset-%02d",i),
    ATSMatchKey:id,
    Axles:[]runtimeAxle{{Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}},
  }
 }
 for i,tr:=range p.Trailers {
  got,reason,ok:=selectedPhysicalTrailerForATS(p,tr.ATSMatchKey)
  if !ok||got.ID!=tr.ID||reason!="ASSOCIATION_AUTOMATIQUE_EXACTE" {
    t.Fatalf("automatic association failed at %d: got %s reason %s ok %v",i,got.ID,reason,ok)
  }
 }
 if _,reason,ok:=selectedPhysicalTrailerForATS(p,"ats.unknown");ok||
    reason!="REMORQUE_INCONNUE_ASSOCIATION_REQUISE" {
  t.Fatalf("unknown trailer must fail closed, got %s %v",reason,ok)
 }
 if _,_,ok:=selectedPhysicalTrailerForATS(p,"");ok {
  t.Fatal("missing telemetry ID must not select anything")
 }
 p.PhysicalSelectedTrailerID=p.Trailers[0].ID
 if _,_,ok:=selectedPhysicalTrailerForATS(p,"ats.unknown");ok {
  t.Fatal("sticky manual preset must not override unknown connected trailer when links exist")
 }
 p.Trailers[1].ATSMatchKey=p.Trailers[0].ATSMatchKey
 if _,reason,ok:=selectedPhysicalTrailerForATS(p,p.Trailers[0].ATSMatchKey);ok||
    reason!="IDENTIFIANT_REMORQUE_AMBIGU" {
  t.Fatalf("ambiguous model mappings must fail closed: %s %v",reason,ok)
 }
}
func TestOriginal14LegacyManualConfigAndSinglePreset(t *testing.T) {
 p:=runtimeProfile{Trailers:[]runtimeTrailer{{ID:"A"},{ID:"B"}}}
 if _,_,ok:=selectedPhysicalTrailerForATS(p,"scs.lowboy");ok {
  t.Fatal("legacy multiple unassociated presets must not guess")
 }
 p.PhysicalSelectedTrailerID="B"
 t2,reason,ok:=selectedPhysicalTrailerForATS(p,"scs.lowboy")
 if !ok||t2.ID!="B"||reason!="SELECTION_MANUELLE_NON_ASSOCIEE" {t.Fatalf("old manual presets regression: %+v %s %v",t2,reason,ok)}
 p.Trailers=p.Trailers[:1]
 p.PhysicalSelectedTrailerID=""
 t1,reason,ok:=selectedPhysicalTrailerForATS(p,"scs.lowboy")
 if !ok||t1.ID!="A"||reason!="SELECTION_MANUELLE_NON_ASSOCIEE" {t.Fatalf("single preset regression: %+v %s %v",t1,reason,ok)}
}
