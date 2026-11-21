# EssieuPilot — étape de validation du braquage physique ATS 1.62

**Branche de recherche uniquement.** Ni release Original.10, ni installateur de DLL, ni mod .scs. Original.9 reste inchangée.

## Pourquoi cette étape

Original.9 récupère les commandes, les profils et la télémétrie des remorques. Mais le SDK SCS fournit des **lectures**, pas une API publique permettant de diriger séparément les essieux de remorque.

Une ancienne implémentation open source MIT (*ts-extra-utilities*) contrôlait la direction d'une **unité de remorque** via des interfaces internes non documentées. Son ancien code utilise une signature binaire de set_individual_steering, un index de fonction virtuelle 73 pour steering_advance, et les anciens offsets 0x04E8 (steering) et 0x04F0 (données de direction).

Ces valeurs **ne sont pas validées** sous ATS 1.62. Une DLL reprenant les anciennes adresses pourrait provoquer un crash ou rester sans effet.

## L'outil EssieuPilot-ControleATS.exe

Cet utilitaire portable Windows ne fait que lire amtrucks.exe, confirmer qu'il s'agit d'un binaire PE x64, obtenir sa version, calculer son SHA-256 et chercher la signature historique avec les octets jokers.

Aucune injection, aucun hook, aucune modification du jeu. Il ne copie pas le binaire du jeu dans le rapport.

Par défaut, il recherche les emplacements Steam courants et bibliothèques additionnelles. On peut aussi lui passer le chemin exact de amtrucks.exe.

Le résultat est enregistré sous :

Documents / EssieuPilot / Diagnostics / Rapport_Compatibilite_ATS.json

Ce fichier peut être partagé pour identifier la version exacte et orienter l'étude ABI. Une signature trouvée n'implique PAS que les offsets ou le vtable historique soient corrects ; ils doivent être confirmés par analyse native et test contrôlé. Une signature absente interdit l'utilisation de cette méthode historique sans adaptation.

## Après réception du rapport

1. Examiner l'empreinte et la présence/unicité de la signature.
2. Vérifier l'ABI réelle pour le binaire précis d'ATS 1.62 : vtable, structures, appels internes.
3. Préparer une DLL native candidate **désactivée par défaut**, liée au canal IPC déjà implémenté, avec arrêt d'urgence.
4. Tester localement en solo la roue, sa télémétrie et la trajectoire ; distinguer accusé de réception IPC et véritable direction physique.
5. Seulement après validation, soumettre une version à autorisation explicite pour publication et installation automatique.

Le contrôle de groupes d'essieux indépendants et le mode Convoi restent hors périmètre.

Sources :
- https://github.com/dariowouters/ts-extra-utilities
- https://modding.scssoft.com/wiki/Documentation/Engine/Units/accessory_chassis_data
