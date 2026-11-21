# ATS 1.62.2.2 — empreinte recue le 21 novembre 2026

## Observations du rapport de l'utilisateur

Source : rapport JSON `Rapport_Compatibilite_ATS.json` fourni apres execution de l'outil de diagnostic en lecture seule.

- Jeu : ATS
- Version du fichier Windows : `1.62.2.2 (7173c707daa083ad5e30a8dda5d5c1a4791a36ca)`
- Taille : `53640592` octets.
- SHA-256 du fichier `amtrucks.exe` : `63ecae85a070934530d9fd971d4d56e5479f1214d7077b403f79b8981c613e08`.
- Signature historique `set_individual_steering` : **1 occurrence**.
- Decalage dans le **fichier brut** : `0x84CD40`. Ce nombre ne doit pas etre confondu avec l'adresse virtuelle ou un RVA sans lecture des sections PE.

## Interpretation

Le seul constat etabli est que la sequence d'instructions caracteristique
du vieux prototype TS Extra apparait une fois dans le binaire actuel.
Cela renforce l'interet de cette piste sans valider sa fonction exacte
ni les structures et appels C++ du moteur.

**NON VALIDES :**
- `vehicle_shared_u::steering` a l'ancien offset `0x04E8`.
- `vehicle_shared_u::wheel_steering_stuff` a l'ancien offset `0x04F0`.
- `steering_advance` a l'ancien index de table virtuelle `73`.
- Parametre normalise, sens de rotation, 35 degres equivalents a une consigne.
- Effets visuels, collision, trajectoire, essieux differencies, mode Convoi.

## Suite

Le verrou de securite R&D `ATS162ExactBuildGate.hpp` impose cette taille,
ces octets de signature et ce SHA-256 exact **avant** l'initialisation du
prototype natif. Un autre binaire est refuse. Le verrou ne rend pas
les offsets historiques surs.

Afin de valider les offsets et l'index vtable, il faudra une analyse plus
complete de l'executable exact, fournie a titre de diagnostic local ou sous
forme d'elements techniques suffisants. Le rapport actuel ne contient
aucune instruction autour des appels ni aucun plan des structures.

Ne pas installer automatiquement la DLL et ne pas publier de mise a jour
physique tant qu'un test sous ATS 1.62 n'a pas demontre un mouvement
reel et une modification de trajectoire, distincts d'une simple commande
IPC recue par la DLL.
