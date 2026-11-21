# EssieuPilot Original.8 — programme R&D du braquage physique ATS

> **Branche de recherche uniquement.** Ne pas distribuer le prototype comme une version fonctionnelle. Original.7 demeure la version installable pour les commandes et diagnostics.

## État technique

- Les commandes clavier, joystick à boutons et joystick analogique restent celles d'Original.7.
- Une DLL C++ expérimentale dérivée de `ts-extra-utilities` (MIT, commit figé `41b307496960732571de8e43100adba75cd115a1`) contient le mécanisme candidat de direction physique **par unité de remorque**, pas indépendamment par essieu.
- La DLL vérifie via SCS SDK la famille de jeu ATS et la version 1.62 avant de publier le nouveau témoin de chargement `Local\\EssieuPilotProbeV8`. Le témoin ne prouve ni la validité des structures internes, ni le mouvement des roues.
- Le contrôleur Go lit le témoin, calcule la consigne analogique/digitale, puis reste **neutre par défaut**.
- Une sortie IPC `enabled=true` n'est accessible **que dans la branche R&D** via la variable d'environnement exacte `ESSIEUPILOT_UNSAFE_ATS162_TEST=I_ACCEPT_CRASH_RISK`, avec toutes les conditions d'autorisation ci-dessous.
- La télémétrie SCS du Hub donne également la rotation de chaque roue via `data.trailer.wheels[].steering`, dans l'unité du SDK (tours). Le diagnostic la convertit en degrés pour la comparer aux consignes. Les mesures vides, sans remorque, ou hors plage sont rejetées ; une valeur nulle ou une mesure SDK n'est pas à elle seule une preuve visuelle.
- Le prototype C++ écrit un accusé de réception `Local\\EssieuPilotAppliedV1` après l'appel expérimental de la fonction interne. Un ACK signifie seulement qu'une fonction C++ a été appelée ; pas que les roues ont bougé physiquement ou visuellement.

## Conditions cumulatives d'envoi d'une consigne R&D

1. Consentement explicite via la variable de recherche.
2. ATS lancé au premier plan et pilotage manuel ACTIF.
3. DLL expérimentale ayant signalé le chargement SDK dans ATS 1.62.
4. Télémétrie SCS récente (moins de 750 ms), non en pause et indiquant **une seule remorque attelée**.
5. Profil actif correspondant à cette remorque (si une clé de correspondance est définie) et contenant **une seule unité de remorque** aux essieux compatibles entre eux.
6. Angle calculé fini et borné, transformé provisoirement en valeur normalisée `[-1,+1]`.

Au moindre échec, le contrôleur publie un paquet IPC neutre `enabled=false` toutes les 40 ms ; les commandes expirent également après 250 ms. Cela **réduit les risques de commandes persistantes, mais ne protège pas contre un crash lié à un offset ou hook incorrect**.

## Ce qui reste obligatoirement à valider dans le vrai jeu

- Confirmer par essai solo que le chargeur SCS initialise la DLL et que la fenêtre de diagnostic indique `CHARGEMENT DLL ATS : CONFIRMÉ`.
- Vérifier sur la version exacte d'ATS les signatures binaires et l'index de méthode virtuelle utilisés par le code ancien. **La validation SCS de la version majeure/mineure ne suffit pas.**
- Seulement ensuite, en essai privé contrôlé avec sauvegardes, vérifier qu'un IPC actif déclenche un ACK récent.
- Comparer l'angle demandé à la télémétrie `trailer.wheel.steering` pour chaque roue ; un ACK avec une télémétrie inchangée serait un indice d'écriture ineffective.
- Enfin, observer et mesurer **le mouvement réel et visuel** des roues, l'effet en conduite, l'angle à 30 % de joystick, le maintien à l'arrêt et le retour au centre en avançant.
- Tester le repli neutre au relâchement, en mode INACTIF, après Alt+Tab, en perte de télémétrie et au dételage.
- Contrôler le modèle de remorque : certains modèles ne possèdent pas d'animation d'essieux directeurs. Une fonction appelée ou un ACK ne garantit pas la prise en charge du modèle.
- Démontrer la stabilité et prévoir désinstallation/restauration avant d'envisager un installateur. **Multijoueur Convoi non validé.**

## Construction automatisée (sans distribution)

Le workflow GitHub `.github/workflows/steering-reference-build.yml` sur `development/original8-physical` compile sous Windows les sources C++ issues du dépôt MIT épinglé, exécute les tests IPC, compile le backend Go et teste les protocoles de chargement/ACK ainsi que la logique de verrouillage. Aucun installateur ni DLL expérimentale ne sont publiés par ce workflow.

**Important :** aucune validation physique dans ATS ne peut être déduite d'un résultat GitHub Actions vert. Tant que les hooks ATS 1.62 n'ont pas été vérifiés dans le jeu, la DLL ne doit pas être ajoutée automatiquement aux installations utilisateur.
