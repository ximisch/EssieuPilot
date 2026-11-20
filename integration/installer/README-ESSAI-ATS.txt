EssieuPilot — Essai physique ATS 1.62 (version 1.7.0-beta.2)
================================================================

IMPORTANT : CECI EST UNE PREVERSION EXPERIMENTALE, PAS UNE VERSION
VALIDEE DE LA DIRECTION PHYSIQUE DANS ATS.

Avant installation :
1. Fermer ATS et Steam si nécessaire.
2. Faire une copie de sécurité de tes sauvegardes ATS.
3. Garder Westplay désactivé (westplay.dll absent du dossier plugins).
4. ATS 1.62, Windows x64, DirectX 11, mode SOLO uniquement.
5. Les trois DLL ETS2LA existantes ne sont PAS supprimées ni remplacées.

Installation (un seul installateur) :
1. Installer EssieuPilot. Cette préversion reste séparée de la version
   officielle 1.6.2. Elle remplace uniquement une ancienne bêta 1.7.
2. Pendant l'installation, cocher :
     "Installer les DLL dans ATS 1.62 (EXPERIMENTAL...)"
   pour autoriser la copie du module physique et du module télémétrie.
   Si tu ne coches pas cette option, le logiciel s'installe SANS DLL ATS.
3. Si ATS est introuvable ou les droits insuffisants, le programme
   affichera une erreur. Ne copie aucune DLL manuellement sans diagnostic.
4. Lancer EssieuPilot AVANT de lancer ATS (important : la DLL n'active pas
   ses fonctions si EssieuPilot ne tourne pas déjà au chargement du jeu).
5. Charger une partie solo en camion, atteler UNE remorque avec essieux
   directeurs. Vérifier que la télémétrie affiche ATS connecté.
6. Par défaut : F8 pour activer/désactiver, F9 gauche, F10 droite.
   Les touches se modifient dans les profils EssieuPilot.
7. ESSAI : vérifier visuellement les roues, rouler lentement et essayer
   un recentrage. Le contrôle est par unité de remorque, PAS par essieu
   indépendant. Angle 35 degres = consigne non étalonnée physiquement.

Désinstallation / récupération en cas de crash :
1. Fermer ATS.
2. Menu Démarrer > EssieuPilot > "EssieuPilot - retirer les DLL ATS".
3. Si le script ne démarre pas ou manque de droits, lancer l'installateur
   avec privilèges admin et utiliser le raccourci.
4. Si ATS ne démarre plus et que le script ne fonctionne pas :
   déplacer uniquement ces deux fichiers du dossier
   ATS\bin\win_x64\plugins :
        EssieuPilotATS.dll
        scs_telemetry_hub.dll
   Ne PAS déplacer scs-telemetry.dll, scs_sdk_controller.dll ou
   ets2la_plugin.dll en pensant qu'EssieuPilot les a installées.
5. Envoyer game.log.txt et game.crash.txt après un crash.

Limitations :
- Le module C++ expérimental utilise des fonctions/structures internes du jeu
  remontant à 2025. Leur compatibilité ATS 1.62 n'est PAS prouvée.
- Il peut faire planter le jeu. EssieuPilot vérifie les commandes reçues,
  mais ne peut pas garantir la sûreté de pointeurs internes obsolètes.
- Aucun support Convoi/TruckersMP n'est annoncé.
- Le plugin ne pilote pas physiquement chaque essieu distinctement.
- Le programme n'est pas signé numériquement. SmartScreen peut avertir.
- Les releases officielles stables restent à la version 1.6.2.
