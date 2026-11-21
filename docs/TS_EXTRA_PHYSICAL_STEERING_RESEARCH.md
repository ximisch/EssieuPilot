# EssieuPilot — preuve de concept pour un braquage physique dans ATS

État au 20 novembre 2026. **Recherche et architecture uniquement, aucun essai sous ATS 1.62 ni validation Convoi.**

## Découverte importante : ts-extra-utilities

Projet open source **MIT** : https://github.com/dariowouters/ts-extra-utilities (dernier binaire publié v1.0.1 le 18 janvier 2025 ; le code source indique un projet expérimental).

Le module `src/windows/trailer_manipulation.cpp` réalise **réellement des commandes de direction manuelle de remorque** :

1. Trouve la fonction interne `set_individual_steering_fn` par recherche de signature mémoire : `48 89 5c 24 08 48 89 74 24 10 57 48 83 ec ? 8b 41 ? 48 8b d9 0f 29 74`.
2. Intercepte `physics_trailer_u::steering_advance` (index de table virtuelle 73) pour laisser la physique d'origine agir, ou bloquer l'actualisation automatique lorsque l'utilisateur verrouille la direction.
3. Écrit un angle normalisé `[-1,+1]` dans `current_trailer->steering` et appelle `set_individual_steering_fn_(current_trailer->wheel_steering_stuff, current_trailer->steering)`.
4. Parcourt la chaîne des remorques `current_trailer->slave_trailer`, offrant ainsi une commande **par unité de remorque**, mais **pas encore démontrée par essieu**.
5. La preuve de concept d'origine possède boutons gauche/droite et curseur ImGui, non configurables au clavier/gamepad selon l'objectif EssieuPilot.

Références exactes :
- https://github.com/dariowouters/ts-extra-utilities/blob/main/src/windows/trailer_manipulation.cpp
- https://github.com/dariowouters/ts-extra-utilities/blob/main/src/prism/vehicles/vehicle_shared.hpp
- https://github.com/dariowouters/ts-extra-utilities/blob/main/src/prism/functions.hpp
- https://github.com/dariowouters/ts-extra-utilities/blob/main/README.md

## Limites de compatibilité critiques

- Le code utilise des **offsets internes fixes** : `vehicle_shared_u::steering` à `0x04E8` et `wheel_steering_stuff` à `0x04F0`, ainsi qu'un index de méthode virtuelle et une signature binaire. Ces valeurs appartiennent à l'époque de développement de ts-extra-utilities (avant ATS 1.62) et **ne sont pas confirmées sur ATS 1.62**.
- Comparaison : le projet récent ETS2LA/plugin présente des structures de remorque différentes (p. ex. `vehicle_shared_u` a un `wheel_steering_stuff` vers `0x04D0` selon un commentaire portant sur 1.54). Les offsets ne peuvent pas être recopiés sans validation.
- Le plug-in dépend **DirectX 11**, est marqué **expérimental / risque de crash**, et son auteur déconseille le multijoueur.
- La fonction agit **par remorque**, sur un angle normalisé. Aucune preuve d'un contrôle physique distinct de chaque essieu ni d'une conversion stricte `35°` -> `1.0`.
- Il existe des allégations publiques concernant la sûreté d'un binaire distribué, non vérifiées : par prudence, **ne pas intégrer ni redistribuer le DLL précompilé tiers**. Utiliser uniquement le code open source audité avec respect de la licence MIT si l'intégration se poursuit.

## Architecture proposée pour EssieuPilot

1. **Bridge de jeu dédié, issu de code source audité** : module SCS chargé dans ATS, permettant de lire un message de direction pour chaque remorque depuis un canal IPC local et d'appliquer ce contrôle sur le thread de jeu. Pas de hook actif par défaut.
2. **Détection et validation de version** : reconnaître ATS x64 et sa version exacte, puis vérifier les signatures, pointeurs, structures, limites et longueur des chaînes de remorques avant d'autoriser le moindre appel interne. Sur les versions inconnues (dont 1.62 tant qu'inconnue), désactiver l'écriture.
3. **Validation observable** : en essai privé solo DirectX 11, tourner à l'arrêt, vérifier `trailer.wheel.steering` par télémétrie, avancer, recentrer, tester plusieurs remorques, désactiver en cas de divergence. Aucun progrès « simulé » ne doit être qualifié de commande réelle.
4. **Moteur de contrôle** : entrée clavier/manette/joystick configurable, 14°/s, retour 7°/s, cap 35° souhaité sous réserve d'étalonnage, profils sauvegardés par remorque (groupes d'essieux à étudier séparément).
5. **Installation / rollback sûrs** : sauvegarde et restauration des DLL et journaux, retrait immédiat sur crash, aucune modification des autres plugins. Westplay doit être retiré par l'utilisateur avant essais.
6. **Convoi** : non garanti, aucune déclaration de compatibilité avant tests en ligne.

**Publication bloquée tant qu'ATS 1.62 ne valide pas au moins la direction physique par remorque.** Cette découverte rend l'objectif plausible mais ne le garantit pas.
