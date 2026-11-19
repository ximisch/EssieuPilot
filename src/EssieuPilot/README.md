# EssieuPilot – édition de test pour les mises à jour

Cette application Windows a été écrite indépendamment de l'EssieuPilot 1.6.1 préexistant, dont le code source n'est pas dans ce dépôt. Elle ne remplace pas la version actuelle.

Fonctions : interface sombre, profils de remorques, nombre d'essieux, inversion, angle maximal, vitesses de braquage/recentrage, simulation au clavier (A/D), recentrage dès que « avance » est coché, sauvegarde JSON, vérification GitHub et mise à jour avec consentement.

Les paramètres restent dans `%APPDATA%\EssieuPilot-Test\profiles.json`. Le programme est installé sous `%LOCALAPPDATA%\Programs\EssieuPilot-Test`, avec un AppId distinct, sans toucher aux jeux et profils existants.

Test de bout en bout : installer `v1.7.0-test.1`, ouvrir « Mises à jour », vérifier puis installer `v1.7.0-test.2`. L'application contrôle l'URL et le SHA-256 publié par GitHub Releases avant l'exécution de l'installateur téléchargé. L'installateur n'est pas signé. Pour tester, accepter l'installation et fermer ATS avant toute intervention sur les plugins (non inclus ici).

**Ce programme simule les essieux uniquement dans sa fenêtre. Aucune DLL ATS, aucune commande physique ni compatibilité Convoi n'est intégrée ou validée.** Il ne doit pas être présenté comme une mise à niveau complète de la version 1.6.1.

## Test de conservation des réglages : v1.7.0-test.2 → v1.7.0-test.3

1. Sur **test.2**, menu « Profils de remorques » : créer ou modifier un profil, par exemple « Vérification sauvegarde », avec un nombre d'essieux inhabituel, une vitesse modifiée et l'option « Inverser la direction » cochée ; cliquer **Enregistrer le profil**.
2. Fermer et rouvrir test.2 pour vérifier que le profil reste enregistré AVANT la mise à jour.
3. Dans « Mises à jour », cliquer « Vérifier les mises à jour », puis installer **test.3** (seulement après publication de la release).
4. Rouvrir EssieuPilot et confirmer la version **1.7.0-test.3**.
5. Retourner dans « Profils de remorques », sélectionner le profil et comparer le nom, le nombre d'essieux, le braquage maximal, la vitesse de braquage, la vitesse de retour et l'inversion de direction.

Le code de test.3 **ne change ni le chemin des profils ni leur format JSON**. Une indication « Profils chargés » apparaît dans l'écran « Mises à jour ». La conservation devra être confirmée sur le PC Windows de l'utilisateur.
