# EssieuPilot

Application Windows de configuration et de simulation des essieux directeurs pour **American Truck Simulator**.

## État du projet

- Application Windows avec sauvegarde de profils, commandes analogiques, réglages par essieu et représentation 2D/3D
- Lecture des données du jeu uniquement avec une DLL SCS compatible et après vérification sur la machine de l'utilisateur
- **Le braquage physique dans ATS et la synchronisation Convoi ne sont pas validés**
- Le centre **Mises à jour** consulte les publications publiques de ce dépôt (`Releases`)
- Pas de mise à jour installée sans accord explicite de l'utilisateur
- Téléchargement accepté uniquement pour `Installer_EssieuPilot.exe` depuis ce dépôt et avec empreinte SHA-256 annoncée par GitHub
- Profils sauvegardés séparément dans `%APPDATA%\EssieuPilot\profiles.json`

## Publication des versions

Les versions publiées doivent être créées en tant que GitHub Release, avec un tag sémantique tel que `v1.6.2` et le fichier Windows **Installer_EssieuPilot.exe** comme ressource de la release. La compilation nécessite le code source complet du logiciel.

Le numéro technique est visible dans **À propos / Mises à jour**, mais le nom affiché reste **EssieuPilot**.

**Important :** le dépôt est actuellement en cours de préparation. Aucun correctif physique ATS ni compatibilité multijoueur n'est à considérer comme fonctionnel sans tests réels. Le fichier téléchargé dans la conversation ChatGPT n'est pas automatiquement publié comme GitHub Release.

## Sécurité

- Vérifier les binaires avant publication, notamment par antivirus.
- Ne jamais publier les données personnelles, profils utilisateurs, secrets, ou clés de signature.
- Compiler et vérifier l'exécutable dans une chaîne de build connue.
- Tester la migration des profils et le retour arrière.
- La vérification SHA-256 assure l'intégrité des fichiers, mais ne remplace pas une signature de code Windows.

## Édition de test avec code source (nouveau)

Pour ne **pas remplacer la version officielle 1.6.1**, une édition de test indépendante est maintenant construite avec son code source complet :

- [Source Windows C# (.NET 8)](src/EssieuPilot/Program.cs)
- [Projet de compilation](src/EssieuPilot/EssieuPilot.csproj)
- [Installeur Inno Setup](installer/EssieuPilot-Test.iss)
- [Compilation et publications automatiques](.github/workflows/test-release.yml)

Après publication, installer la préversion `v1.7.0-test.1`, puis utiliser le bouton « Mises à jour » pour passer à `v1.7.0-test.2`. Le test utilise ses **propres dossiers** et sauvegarde ses profils séparément.

**Limite :** cette édition est une reconstruction de test et non la source historique du programme 1.6.1 ; elle ne pilote pas physiquement les essieux d'ATS. Aucun fonctionnement en Convoi n'est démontré. Les exécutables doivent encore être testés sur un véritable PC Windows.
