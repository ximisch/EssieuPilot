# EssieuPilot — étape 1 : détection du processus ATS

## Confirmé sur le PC de l'utilisateur

Windows détecte American Truck Simulator via **`amtrucks.exe`**. Le test PowerShell a montré les transitions `ATS NON LANCÉ` → `ATS LANCÉ` sans aucun composant de télémétrie.

## Intégration de développement

Le fichier `apply_detection.py` ajoute aux **sources reconstruites de développement uniquement** :

- Route `GET /api/ats/process` renvoyant `{"running":true/false,"process":"amtrucks.exe"}`, avec `Cache-Control: no-store`.
- Sondage du processus toutes les secondes dans l'interface récupérée, sans utiliser Steam, la télémétrie ou une DLL ATS.
- Affichage `ATS LANCÉ`, `ATS NON LANCÉ` ou `ÉTAT INDISPONIBLE` en cas d'erreur de communication ; la télémétrie reste un état séparé.
- Tests Go avec processus simulé présent/absent, sans lancer ATS.

La logique Windows `gameProcessRunning()` utilise l'API de lecture seule Toolhelp32 (énumération des processus) déjà présente dans la branche de développement. Un appel à `Get-Process` donne le même principe sans modifier le jeu.

## Ce que cela ne fait pas

- **Ne change pas la 1.6.2 stable installée.** L'exécutable historique contient les ressources de l'interface, mais le **code serveur Go historique complet n'est pas dans le dépôt**. Il ne serait pas fiable de patcher son binaire.
- **Ne publie aucune version et n'installe rien sur le PC.**
- Ne détecte pas les remorques, les essieux ou le braquage physique, ni les événements d'une session Convoi.
- La compilation et les tests CI prouvent la cohérence de cette intégration dans les sources reconstruites ; **l'affichage dans la fenêtre historique de l'utilisateur n'est pas encore validé**.

## Conditions pour une mise à jour 1.6.2 non destructive

Le code Go original complet de l'application stable, ou un point d'extension explicitement prévu par celle-ci, serait nécessaire pour ajouter cet affichage à la fenêtre actuelle **sans reconstruire ni remplacer le launcher**.

Ne pas annoncer une release officielle 1.6.3 avant test réel de l'application et de son mécanisme de mise à jour intégré.
