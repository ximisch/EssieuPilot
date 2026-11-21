# EssieuPilot — chantier Original.9 (base technique)

Ce dossier contient un **pont de télémétrie et un installateur de modules vérifiés** pour ATS.
Il ne remplace PAS EssieuPilot Original.8, sa fenêtre, son système de mise à jour
ou ses profils officiels. Cette base est en développement et ne commande PAS
physiquement les roues de remorque.

## Fonctions réellement implémentées dans cette branche

- Détection du processus Windows \`amtrucks.exe\` ;
- Lecture optionnelle de la télémétrie via un hub local disponible sur
  \`http://127.0.0.1:38471/snapshot\`, si ce hub est **installé, en cours
  d'exécution et compatible** (le programme ne télécharge PAS le hub) ;
- Contrôle que le JSON correspond bien à ATS, lecture de \`trailer.connected\`,
  de \`trailer_spec.id\`, du nom et de la vitesse ;
- Estimation du nombre d'essieux en regroupant les positions Z des roues ;
  ce nombre n'est PAS une preuve que les essieux sont directeurs ;
- Profil reconnu uniquement si un lien explicite \`id remorque -> nom du profil\`
  a été enregistré (aucune création de profil directeur pour une remorque inconnue) ;
- Statut lisible dans \`%LOCALAPPDATA%\EssieuPilot\Original9\bridge-status.json\`,
  mis à jour à chaque interrogation ;
- Déploiement de binaires exclusivement présents dans un paquet local explicitement
  fourni avec manifeste et SHA-256 valides, et refus d'écraser des modules tiers ;
- Sauvegarde avant remplacement et tentative de retour arrière si l'installation échoue ;
- Refus d'installer quoi que ce soit pendant que ATS tourne ;
- Tests autonomes de lecture télémétrique.

## Compilation (pour développeurs)

\`\`\`powershell
dotnet build src/EssieuPilot.Original9Bridge/EssieuPilot.Original9Bridge.csproj -c Release
dotnet run --project src/EssieuPilot.Original9Bridge -- self-test
\`\`\`

Sur Windows, \`EssieuPilotBridge.exe status\` lit une fois l'état ATS.
\`EssieuPilotBridge.exe watch\` actualise l'état continuellement (Ctrl+C pour arrêter).
Le profil peut être associé à l'identifiant exact de la remorque :

\`\`\`powershell
EssieuPilotBridge.exe profiles add "scs.lowboy" "Mon profil lowboy" 3
\`\`\`

Le nombre d'essieux est une déclaration de l'utilisateur et ne prouve pas que
la remorque possède des essieux directeurs. La télémétrie n'actionne aucune roue.

## Installateur de modules

Il ne contient volontairement **aucune vraie DLL de commande physique** :
aucun hook de braquage vérifié n'existe dans les sources publiées d'EssieuPilot.

Un paquet validé localement peut contenir :

\`\`\`text
bundle/
  modules.json
  scs_telemetry_hub.dll
  EssieuPilot_MonMod.scs
\`\`\`

Exemple de manifeste (les empreintes ci-dessous doivent être remplacées par les
véritables 64 caractères hexadécimaux calculés sur les fichiers approuvés) :

\`\`\`json
{
  "modules": [
    { "file": "scs_telemetry_hub.dll", "target": "plugin", "sha256": "<SHA256_HEX_64>" },
    { "file": "EssieuPilot_MonMod.scs", "target": "mod", "sha256": "<SHA256_HEX_64>" }
  ]
}
\`\`\`

Commande explicite :

\`\`\`powershell
EssieuPilotBridge.exe install-modules "D:\Steam\steamapps\common\American Truck Simulator" "C:\EssieuPilot\bundle" "C:\EssieuPilot\bundle\modules.json"
\`\`\`

La commande contrôle que \`amtrucks.exe\` existe dans le chemin ATS sélectionné,
vérifie TOUTES les empreintes avant de copier, refuse les noms inconnus,
ne remplace pas de DLL tierces, conserve des sauvegardes dans
\`%LOCALAPPDATA%\EssieuPilot\Original9\backups\` et restaure les fichiers
si la transaction échoue. L'installation ne suffit pas à activer les mods
\`.scs\` : le gestionnaire de mods ATS doit encore les activer.

Les hashes SHA-256 prouvent l'intégrité, pas la confiance du fournisseur ;
les DLL téléchargées doivent être vérifiées et leur licence respectée.

## Limitations impératives

1. **Le code source de l'interface originale Original.8 n'est pas dans ce dépôt.**
   Cette branche n'intègre donc pas ses fonctionnalités au bouton « Mises à jour ».
2. **Le SCS Telemetry SDK lit les données du jeu ; il ne fournit pas d'API
   publique documentée pour commander librement chaque essieu arrière.**
   Le contrôle manuel indépendant des groupes, les 35°, 14°/s et 7°/s
   sont des exigences, PAS des fonctionnalités physiques implantées.
3. Pas de hook mémoire ni d'injection non testée. Aucun mode Convoi validé.
4. Pas de paquet de modules fourni : en particulier pas de DLL physique ni de
   mod \`.scs\` publié comme solution de braquage.
5. Avant une Original.9 distribuée officiellement, il faudra récupérer les
   sources de l'interface originale, vérifier l'intégration réelle ATS 1.62,
   mesurer le comportement des essieux dans le jeu, faire passer les tests
   de compilation Windows et effectuer des essais sur un PC de jeu réel.

Source du schéma JSON du hub de télémétrie :
https://github.com/UncappingBadger/Badgers-SCS-Telemetry-Hub

L'ancien Original.8 et ses profils restent intacts.
