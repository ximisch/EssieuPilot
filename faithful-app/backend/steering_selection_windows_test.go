//go:build windows
package main
import "testing"
func TestOriginal13GenericSelectedPhysicalTrailer(t *testing.T) {
 base:=runtimeProfile{Trailers:make([]runtimeTrailer,20)}
 for i:=range base.Trailers {
   base.Trailers[i]=runtimeTrailer{ID:string(rune('A'+i)), Axles:[]runtimeAxle{
    {Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1},
   }}
 }
 if _,ok:=selectedPhysicalTrailer(base);ok{t.Fatal("multiple presets must require explicit selection")}
 for i:=range base.Trailers {
   base.PhysicalSelectedTrailerID=base.Trailers[i].ID
   chosen,ok:=selectedPhysicalTrailer(base)
   if !ok || chosen.ID!=base.Trailers[i].ID{t.Fatalf("arbitrary trailer %d incorrectly selected",i)}
   axle,compatible:=trailerCompatible(chosen)
   if !compatible || axle.MaxAngle!=35{t.Fatalf("valid selected preset %d incompatible",i)}
 }
 base.PhysicalSelectedTrailerID="does-not-exist"
 if _,ok:=selectedPhysicalTrailer(base);ok{t.Fatal("unknown selected trailer silently changed")}
 base.Trailers=base.Trailers[:1]
 base.PhysicalSelectedTrailerID=""
 if x,ok:=selectedPhysicalTrailer(base);!ok||x.ID!="A"{t.Fatal("single preset should be usable without extra click")}
}
func TestOriginal13RejectOnlyNonUniformSelectedAxles(t *testing.T){
 p:=runtimeProfile{Trailers:[]runtimeTrailer{
  {ID:"one",Axles:[]runtimeAxle{
   {Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1},
   {Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}}},
  {ID:"other",Axles:[]runtimeAxle{
   {Enabled:true,MaxAngle:60,TurnRate:14,CenterRate:7,Ratio:1,Reversed:true},
   {Enabled:true,MaxAngle:35,TurnRate:14,CenterRate:7,Ratio:1}}},
 }}
 p.PhysicalSelectedTrailerID="one"
 tr,ok:=selectedPhysicalTrailer(p)
 if !ok {t.Fatal("should select first physical config")}
 if _,compatible:=trailerCompatible(tr);!compatible {t.Fatal("other incompatible stored configs must never block selected valid config")}
 p.PhysicalSelectedTrailerID="other"
 tr,ok=selectedPhysicalTrailer(p)
 if !ok {t.Fatal("should still select other config")}
 if _,compatible:=trailerCompatible(tr);compatible {t.Fatal("different per-axle parameters cannot use scalar native command safely")}
}
