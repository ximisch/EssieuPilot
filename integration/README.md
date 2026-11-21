# EssieuPilot — branche de recherche du contrôle physique ATS

**Ce répertoire est du code source de développement, pas un mod ATS prêt à installer.**

## Ce qui existe réellement

- `bridge/EssieuPilotBridge.hpp` lit un message de direction par remorque sur une mémoire partagée locale (`Local\\EssieuPilotSteeringV1`), avec version de protocole, limites d'angles et expiration après 250 ms.
- `bridge/patch_upstream.py` adapte les sources du projet **TS Extra Utilities** (licence MIT, commit amont épinglé `41b307496960732571de8e43100adba75cd115a1`) pour transmettre ce message au mécanisme expérimental de braquage des roues d'une remorque. Les commandes périmées réactivent le fonctionnement automatique d'ATS.
- `go_sources.part01.b64` à `go_sources.part03.b64` contiennent une archive tar.xz des ajouts Go : calcul des angles, configuration clavier/manette/joystick, mémoire partagée Windows et tests. SHA-256 exact après concaténation et décodage : `bb19be39dad57492951f20b0127de509452bbf3047f99c1bcc931c1a4e3bc1e1`.
- `bridge/patch_go_backend.py` raccorde les sources Go au backend reconstruit sans toucher à la version stable 1.6.2.
- Les GitHub Actions `steering-reference-build.yml` et `ats-integration-dev.yml` compilent et testent les composants Windows mais **ne publient pas d'installateur**.

## Comment fonctionne le lien

La DLL ne reçoit une commande que si l'application est ouverte, le processus ATS se trouve au premier plan, une télémétrie valide signale une remorque et l'utilisateur a activé le pilotage via sa touche configurée. Chaque commande indique un indice de remorque, un angle normalisé [-1,1] et un compteur de fraîcheur ; elle expire en 250 ms. Les contrôles utilisent une simulation locale à 14°/s, retour 7°/s et max 35° par défaut. Ce sont des consignes, **pas encore des mesures d'angle physique confirmées**.

Le prototype issu de TS Extra pilote **une unité de remorque** et non chacun des essieux séparément. Quand plusieurs essieux d'un même profil demandent des inversions/répartitions incompatibles, le contrôleur Go refuse le pilotage physique plutôt que de simuler leur séparation.

## Limitations bloquantes avant toute publication

1. **Offsets et signatures du jeu 1.62** : l'amont TS Extra date de 2025, et ses index de vtable et champs internes n'ont pas été validés sur ATS 1.62. Des crashs sont possibles même si la DLL compile.
2. **Mécanique réelle** : les sources TS Extra indiquent le mécanisme mais aucun test direct dans ATS de l'utilisateur n'a été effectué. Il faut démontrer les angles par télémétrie et en observant les roues.
3. **Par essieu vs par remorque** : la séparation physique des groupes d'essieux directeurs n'est pas démontrée.
4. **Distribution** : préparer un installateur signé/contrôlé avec DLL, licence MIT, migration de profils, détection de conflits et rollback. Ne pas déposer les DLL de test dans `plugins` avant validation.
5. **Convoi** : non validé, à ne jamais annoncer comme compatible.

**Ne pas fournir ce prototype à l'utilisateur comme version complète.** Le travail peut continuer dans cette branche sans changer la release officielle.


## Validation complémentaire (Original.7)

Avant toute diffusion de la DLL dans l'installateur EssieuPilot, la compilation Windows du projet TS Extra modifié, le protocole IPC et la compatibilité des interfaces doivent être contrôlés. Une compilation réussie NE valide PAS les offsets du moteur de physique ATS 1.62, l'index de vtable 73, le bon comportement des essieux ni la stabilité du jeu.

Critères incontournables pour le passage à un test physique : ATS 1.62 exact, fonctionnement en solo, sauvegarde des dossiers du jeu, désactivation de la commande lorsque le mode manuel est INACTIF, délai d'expiration IPC 250 ms, contrôle du mouvement visible, possibilité de revenir en arrière. Tant que les offsets ne sont pas vérifiés sur l'exécutable de cette version, ne pas embarquer automatiquement la DLL expérimentale.
