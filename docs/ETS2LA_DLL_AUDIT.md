# Audit statique des DLL ETS2LA reçues — EssieuPilot

Date de l'analyse : 20 novembre 2026. Analyse **statique seulement** (fichiers PE x64 ; aucun chargement ou exécution des DLL).

## DLL locales fournies par l'utilisateur

| DLL | Taille | SHA-256 | Exportations importantes |
|---|---:|---|---|
| scs_sdk_controller.dll | ~35 Kio | `d9fbd5ace5e76eed70b7e10b78c450b111566ad6aa814dff65d6244ad23e687e` | `scs_input_init`, `scs_input_shutdown` |
| scs-telemetry.dll | ~43 Kio | `f361894a5434746933dab47ed8bc17a6f22f65802ce92256fc7cd7d4fc7fac3a` | `scs_telemetry_init`, `scs_telemetry_shutdown` |
| ets2la_plugin.dll | ~357 Kio | `52e8f78b078a588c816794a18a55815028c0b50c015166d12791c1c0d64e03fe` | `scs_telemetry_init`, `scs_telemetry_shutdown` |

## Observations prouvées

1. **Controller : commandes du jeu, pas d'essieu directionnel indépendant.** La table des commandes contient `steering`, `trailerbrake`, `liftaxle`, `liftaxlet`, `slideaxle...` : la commande `steering` concerne la direction du camion. Aucun nom de commande explicite équivalent à `trailer_steering_angle` / `trailer_axle_steer` n'a été trouvé. Déclaration des commandes : https://github.com/ETS2LA/scs-sdk-controller/blob/main/inputs.h.

2. **Telemetry : lecture des propriétés des roues et remorques.** Les chaînes de la DLL incluent `wheel.steerable`, `trailer.connected`, `trailer.wheel.steering` et `truck.wheel.steering`. **Ce sont des canaux de lecture** ; ils peuvent aider à mesurer l'angle observé et vérifier si un jour une commande physique produit un effet, mais ne permettent pas en eux-mêmes de commander la physique.

3. **ETS2LA plugin : accès mémoire interne du jeu.** Les chaînes incluent `game_trailer_actor` et `controlling steering.`. Le code source MIT à https://github.com/ETS2LA/plugin/blob/main/src/prism/vehicles/game_trailer_actor.cpp recherche la structure d'une remorque ; https://github.com/ETS2LA/plugin/blob/main/src/prism/vehicles/vehicle_shared.hpp expose des structures de roues. La méthode `vehicle_shared_u::set_steering_angle` écrit dans une structure interne : https://github.com/ETS2LA/plugin/blob/main/src/prism/vehicles/vehicle_shared.cpp. **Mais** l'utilisation réelle dans https://github.com/ETS2LA/plugin/blob/main/src/core.cpp cible explicitement `game_actor->game_physics_vehicle` (le **camion**), pas `game_trailer_actor` ni un essieu de remorque. Le code ne valide donc pas l'objectif EssieuPilot.

4. **Risque de crash et de compatibilité de version.** La modification de structures internes du jeu est fragile face aux versions ATS. Un signalement public relatif à `scs-telemetry.dll` et ETS2 1.61 décrit un crash `C0000005`, https://github.com/ETS2LA/ETS2LA/issues/653. Cela **ne prouve pas** que la DLL présente dans cette archive cause les crashs ATS 1.62 de l'utilisateur, mais justifie un test d'isolation de plugins et une version contrôlée. **Ne pas déposer automatiquement plusieurs DLL de télémétrie concurrentes** sans vérifier la compatibilité.

## Plan technique recommandé

- Intégrer la **lecture** de télémétrie (identité ATS, attelage, roue directrice, angle observé, vitesse) avec une seule DLL validée, en gérant les conflits de plugins et le rollback.
- Vérifier si l'angle de chaque roue de la remorque est réellement exposé et utilisable via le SDK sur ATS 1.62 ; documenter la précision et les limites par mod.
- Réaliser un prototype **isolé et réversible** de direction de remorque avant toute intégration en release, avec examen des licences et des risques. La méthode ETS2LA qui écrit l'angle du camion ne peut pas être recopiée comme un correctif d'essieu de remorque.
- N'afficher `Contrôle physique actif` qu'après mesure d'une variation effective des roues en jeu, et conserver `physical_control:false` sinon.
- N'annoncer aucune compatibilité Convoi sans essais.
- Ne publier aucune version complète tant que le contrôle d'essieux indépendants n'a pas été démontré.

## Notes de licence

Les dépôts sources actuels `ETS2LA/plugin` (MIT) et `ETS2LA/telemetry` (GPL-3.0) n'ont pas le même régime de licence. **Ne pas redistribuer les DLL reçues** sans identifier précisément leur origine, leur version et leurs obligations de licence.
