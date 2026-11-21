# Étude statique du binaire ATS 1.62.2.2 — direction de remorque

**Source :** fichier `amtrucks.exe` fourni dans le chat le 21 novembre 2026. **Seuls les résultats techniques sont conservés dans le dépôt : le binaire du jeu n'y est pas copié.**

## Identification du binaire

| Élément | Résultat |
|---|---|
| Version | `1.62.2.2 (7173c707daa083ad5e30a8dda5d5c1a4791a36ca)` |
| Longueur | `53 640 592` octets |
| SHA-256 | `63ecae85a070934530d9fd971d4d56e5479f1214d7077b403f79b8981c613e08` |
| Format | PE32+ x64, base de l'image `0x140000000` |
| Section de code | `.text`, offset fichier `0x400`, RVA `0x1000` |

## Fonction de direction retrouvée

La signature TS Extra `48 89 5C 24 08 48 89 74 24 10 57 48 83 EC ?? 8B 41 ?? 48 8B D9 0F 29 74` apparaît **une seule fois**.

- Offset de fichier : `0x84CD40`
- RVA : `0x84D940`
- VA statique avant ASLR : `0x14084D940`
- Bornes de fonction d'après `.pdata` : RVA `0x84D940–0x84DBC7`
- Cinq appels directs `E8` identifiés aux VA `0x14063F0A2`, `0x140689B0C`, `0x14077B2C9`, `0x140946B73` et `0x140F4C90A`.

**Précaution :** un offset de fichier, une RVA et une VA sont différents. Il ne faut pas prendre `0x84CD40` pour un pointeur dans le jeu.

## Nouveaux champs observés dans les fonctions appelantes

La fonction située à `0x14077B140` prépare le calcul de direction avec les instructions :

```asm
movq  0x6c0(%rsi), %rcx
movaps %xmm2, %xmm1
movss %xmm2, 0x6b8(%rsi)
callq 0x14084d940
```

Une seconde fonction, `0x140F4C580`, lit également la valeur à `+0x6B8` et passe le pointeur à `+0x6C0` à la même fonction.

| Champ candidat | Ancien TS Extra | Accès dans ATS actuel |
|---|---|---|
| Angle de direction | `0x4E8` | `0x6B8` |
| Pointeur de données de direction | `0x4F0` | `0x6C0` |

La fonction `0x14084D940` accède elle-même jusqu'à l'offset `+0x16C` dans l'objet qui lui est transmis ; le modèle historique TS Extra pour ces données faisait seulement `0x60` octets. **Ne pas le copier sans revoir les structures.**

## Candidat pour la nouvelle méthode virtuelle

Dans **trois tables** de pointeurs de fonctions référencées par du code :
- L'entrée **78** correspond à la fonction `0x14077B140`.
- L'entrée **73** correspond à une autre fonction `0x14085C0F0`.

Débuts de table candidats (offsets de fichier) : `0x22D4C88`, `0x22F8D50` et `0x23313D8`. Chaque début possède des références `LEA RIP` dans le code.

Cela suggère fortement que l'ancien index **73** de `steering_advance` est devenu **78**, mais **nous n'avons pas encore identifié avec certitude le type dynamique propriétaire de chaque table ni la signature ABI exacte**. Aucun hook à cet index n'est autorisé à ce stade.

## Constantes utiles observées

- `0x142573278` : `1.0f`
- `0x14257446C` : `-1.0f`
- `0x142573D2C` : `57.29578018f`
- `0x142572C3C` : `0.01745329238f`

Ces constantes sont cohérentes avec une conversion degrés ↔ radians et des limites normalisées ; **elles ne déterminent pas à elles seules la correspondance d'une consigne de 35° avec le braquage réel des roues**.

## Protection obligatoire

Le prototype `integration/bridge/ATS162ExactBuildGate.hpp` comprend les offsets candidats à titre informatif, et un **verrou en dur** :

`ABI_VALIDATED_FOR_WRITES = false`

Tant que ce verrou est faux, `ensure_bridge_hook()` refuse de créer un hook, même si le fichier ATS correspond exactement au SHA-256 connu et si un paquet IPC arrive. Un test C++ statique interdit de réactiver accidentellement ce booléen en construction R&D.

**Statut :** analyse statique réelle ; aucune vérification de l'ABI à l'exécution, aucun mouvement physique démontré, pas de DLL à distribuer, pas de mise à jour Original.10, pas de garantie Convoi.

La suite doit vérifier la table virtuelle, la représentation des données et l'appel en conditions réelles avant toute écriture dans le moteur.