package main

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "time"
)

// Test releases form a separate channel. Never install the RC3 DLL package,
// beta releases or the official 1.6.2 installer as a native update.
const originalUpdateRepo = "ximisch/EssieuPilot"
var originalReleaseTag = regexp.MustCompile(`^v2[.]0[.]0-original[.](\d+)$`)
var originalDigest = regexp.MustCompile(`^sha256:[0-9a-fA-F]{64}$`)

type originalAsset struct {
    Name string `json:"name"`
    URL string `json:"browser_download_url"`
    Digest string `json:"digest"`
}
type originalRelease struct {
    Tag string `json:"tag_name"`
    Name string `json:"name"`
    Body string `json:"body"`
    Prerelease bool `json:"prerelease"`
    Draft bool `json:"draft"`
    Assets []originalAsset `json:"assets"`
}
type originalCandidate struct {
    Version int
    Tag string
    URL string
    Digest string
    Notes string
}

func selectOriginalUpdate(releases []originalRelease, installed int) *originalCandidate {
    var latest *originalCandidate
    for _, rel := range releases {
        if rel.Draft || !rel.Prerelease {continue}
        m := originalReleaseTag.FindStringSubmatch(rel.Tag)
        if m == nil {continue}
        version, err := strconv.Atoi(m[1])
        if err != nil || version <= installed ||
            latest != nil && version <= latest.Version {continue}
        for _, asset := range rel.Assets {
            expected := fmt.Sprintf("https://github.com/%s/releases/download/%s/Installer_EssieuPilot.exe",originalUpdateRepo,rel.Tag)
            if asset.Name != "Installer_EssieuPilot.exe" || asset.URL != expected ||
                !originalDigest.MatchString(asset.Digest) {continue}
            notes:=rel.Body
            if len(notes)>500 {notes=notes[:500]}
            latest=&originalCandidate{Version:version,Tag:rel.Tag,
                URL:asset.URL,Digest:strings.ToLower(asset.Digest[7:]),Notes:notes}
            break
        }
    }
    return latest
}

func currentOriginalBuild() int { return 12 }

func checkOriginalUpdate(ctx context.Context) (*originalCandidate,error) {
    ctx,cancel:=context.WithTimeout(ctx,20*time.Second)
    defer cancel()
    req,err:=http.NewRequestWithContext(ctx,"GET",
       "https://api.github.com/repos/"+originalUpdateRepo+"/releases?per_page=70",nil)
    if err!=nil{return nil,err}
    req.Header.Set("Accept","application/vnd.github+json")
    req.Header.Set("User-Agent","EssieuPilot-original-UI-test")
    client:=http.Client{Timeout:20*time.Second}
    res,err:=client.Do(req)
    if err!=nil{return nil,err}
    defer res.Body.Close()
    if res.StatusCode!=200{return nil,fmt.Errorf("GitHub HTTP %d",res.StatusCode)}
    var releases []originalRelease
    if err:=json.NewDecoder(io.LimitReader(res.Body,3<<20)).Decode(&releases);err!=nil{return nil,err}
    return selectOriginalUpdate(releases,currentOriginalBuild()),nil
}

func (s *server) getOriginalUpdates(w http.ResponseWriter,r *http.Request) {
    latest,err:=checkOriginalUpdate(r.Context())
    if err!=nil {
        jsonReply(w,200,map[string]any{
            "version":technicalVersion,"feed_configured":true,
            "available":false,"download_ready":false,
            "message":"Recherche indisponible : "+err.Error(),
        })
        return
    }
    if latest==nil {
        jsonReply(w,200,map[string]any{
            "version":technicalVersion,"feed_configured":true,
            "available":false,"download_ready":false,
            "message":"Aucune mise à jour de test plus récente pour cette interface.",
        })
        return
    }
    jsonReply(w,200,map[string]any{
        "version":technicalVersion,"feed_configured":true,
        "available":true,"download_ready":true,
        "latest_version":latest.Tag,"notes":latest.Notes,
        "message":"Mise à jour vérifiée disponible : "+latest.Tag,
    })
}

func downloadOriginalInstaller(ctx context.Context, candidate originalCandidate) (string,error) {
    ctx,cancel:=context.WithTimeout(ctx,5*time.Minute)
    defer cancel()
    req,err:=http.NewRequestWithContext(ctx,"GET",candidate.URL,nil)
    if err!=nil{return "",err}
    req.Header.Set("User-Agent","EssieuPilot-original-UI-test")
    client:=http.Client{Timeout:5*time.Minute}
    res,err:=client.Do(req)
    if err!=nil{return "",err}
    defer res.Body.Close()
    if res.StatusCode!=200{return "",fmt.Errorf("Téléchargement HTTP %d",res.StatusCode)}
    if res.ContentLength>200<<20{return "",errors.New("Installateur trop volumineux")}
    path:=filepath.Join(os.TempDir(),fmt.Sprintf("EssieuPilot-Update-%d.exe",time.Now().UnixNano()))
    f,err:=os.OpenFile(path,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
    if err!=nil{return "",err}
    h:=sha256.New()
    written,err:=io.Copy(io.MultiWriter(f,h),io.LimitReader(res.Body,200<<20+1))
    closeErr:=f.Close()
    if err==nil {err=closeErr}
    if err!=nil || written<500000 || written>200<<20 ||
        !strings.EqualFold(hex.EncodeToString(h.Sum(nil)),candidate.Digest) {
        os.Remove(path)
        return "",errors.New("Échec de téléchargement ou empreinte SHA-256 invalide")
    }
    return path,nil
}

func (s *server) installOriginalUpdate(w http.ResponseWriter,r *http.Request) {
    if !s.authorized(r){jsonError(w,403,"Installation refusée : jeton manquant");return}
    candidate,err:=checkOriginalUpdate(r.Context())
    if err!=nil{jsonError(w,502,"Vérification GitHub impossible : "+err.Error());return}
    if candidate==nil{jsonError(w,409,"Aucune mise à jour disponible");return}
    path,err:=downloadOriginalInstaller(context.Background(),*candidate)
    if err!=nil{jsonError(w,502,err.Error());return}
    installer:=exec.Command(path)
    if err:=installer.Start();err!=nil{
        os.Remove(path)
        jsonError(w,500,"Impossible de lancer l'installateur : "+err.Error());return
    }
    jsonReply(w,200,map[string]any{"ok":true,"message":"Installateur lancé. Confirme son exécution dans Windows ; les profils restent enregistrés."})
    go func(){
        time.Sleep(2*time.Second)
        _ = s.http.Close()
    }()
}
