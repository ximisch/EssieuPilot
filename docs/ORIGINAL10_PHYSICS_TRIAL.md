# EssieuPilot Original.10 — essai contrôlé de la DLL physique ATS 1.62.2.2

## Statut

**PRÉVERSION DE RECHERCHE.** Une DLL x64 `EssieuPilotATS.dll` a été
compilée sur Windows et ses exports SDK SCS ont été vérifiés. Le braquage réel
dans le jeu n'a PAS été testé ni garanti. Risque de crash du jeu.

La mise à jour est isolée de la version stable 1.6.2, conserve l'interface
Original.9 et les profils existants, et ajoute un nouveau paquet DLL.

## Compatibilité autorisée

- Binaire exact `amtrucks.exe` ATS 1.62.2.2.
- Taille : 53 640 592 octets.
- SHA-256 : `63ecae85a070934530d9fd971d4d56e5479f1214d7077b403f79b8981c613e08`.
- Un seul ensemble de remorque attelée dans la télémétrie.
- Le profil EssieuPilot sélectionné doit correspondre à la remorque.
- Réglages des essieux compatibles avec une **commande par remorque**.
- Test **SOLO uniquement**. Convoi / TruckersMP non supportés.

## Installation automatique

1. **Sauvegarder** les profils et sauvegardes ATS dans
   `Documents\American Truck Simulator` (y compris `profiles` et `steam_profiles`
   selon le compte).
2. **Fermer ATS**. Original.9 → Mises à jour → Vérifier → Installer Original.10.
3. Dans la fenêtre d'installation, garder l'option
   « Installer automatiquement la DLL expérimentale pour ATS 1.62.2.2 »
   cochée. La DLL est vérifiée et copiée dans
   `<steam>\steamapps\common\American Truck Simulator\bin\win_x64\plugins\EssieuPilotATS.dll`.
4. L'option « Autoriser le braquage physique expérimental (SOLO uniquement,
   risque de crash) » est **décochée par défaut**. Ne la cocher que pour
   autoriser cet essai ; sans cette case, la DLL reste inactive.
5. Lancer EssieuPilot, puis ATS **en solo** avec une remorque dotée de vrais
   essieux directeurs. Activer le mode manuel dans EssieuPilot et utiliser
   les touches configurées.

## Liste des vérifications

- ATS détecté, télémétrie fraîche et remorque reconnue.
- Diagnostic SDK : DLL détectée / version ATS reconnue.
- Le profil actif possède une configuration valide et une seule unité de remorque.
- Mode manuel ACTIF, appui gauche/droite autorisé ; paquet IPC émis.
- Accusé callback DLL éventuellement reçu. Cela ne prouve **pas** la physique !
- À l'arrêt, observer les roues arrière sur caméra extérieure et vérifier
  le déplacement physique/visuel. À petite vitesse sur une zone dégagée,
  vérifier l'orientation de la remorque, puis le recentrage.
- En cas de plantage, ne pas insister : retirer la DLL.

## Revenir à Original.9 / retirer la DLL

Dans le menu Démarrer EssieuPilot, choisir « EssieuPilot - retirer la DLL
physique ». Cela ferme l'essai, retire **uniquement** la DLL dont le
SHA-256 est enregistré et supprime le consentement ; tous les autres
plugins restent inchangés. ATS doit être fermé.

Si le binaire ATS a changé après une mise à jour Steam, le retrait reste
possible (l'exactitude du SHA ATS n'est exigée que pour une installation).

L'installateur ne remplace pas une DLL `EssieuPilotATS.dll` déjà présente
mais non gérée ou modifiée. Dans ce cas, l'installation de la DLL échoue
sans toucher aux autres plugins.

Si ATS plante, les journaux utiles sont `Documents\American Truck Simulator\game.log.txt`
et le diagnostic d'EssieuPilot.

## Convoi

Les mods `.scs` sont reconnus par les sessions Convoi depuis ATS 1.42,
mais un **hook DLL local n'est pas un protocole réseau**. La direction
commandée localement n'est pas prouvée synchronisée chez les autres.
Cette préversion ne crée donc pas de `.scs` factice.

La DLL inhibe l'application physique si le canal SCS
`multiplayer.time.offset` rapporte un décalage non nul. **Ce n'est pas
une détection fiable de toutes les sessions Convoi.**
Jusqu'à des tests de réseau et une synchronisation fondée, **ne pas
utiliser le braquage expérimental en Convoi**.

## Contraintes techniques restantes

L'ancienne source TS Extra utilisait une structure à `+0x4E8/+0x4F0`
et un slot virtuel 73. Sur ce binaire ATS 1.62.2.2, les candidats sont
`+0x6B8/+0x6C0` et slot 78, plus une fonction d'application des roues à
RVA `0x84D940`. Ces correspondances reposent sur du désassemblage
statique. Le nouveau plugin effectue des validations limitées de
pointeurs et vtables avant écriture, mais aucun test réel du jeu n'a pu
valider le modèle d'objet, le rendu des roues ou l'effet sur la trajectoire.

État : **essai technique, pas preuve de bon fonctionnement physique.**
