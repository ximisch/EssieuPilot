# EssieuPilot – édition de test pour les mises à jour

Cette application Windows a été écrite indépendamment de l'EssieuPilot 1.6.1 préexistant, dont le code source n'est pas dans ce dépôt. Elle ne remplace pas la version actuelle.

Fonctions : interface sombre, profils de remorques, nombre d'essieux, inversion, angle maximal, vitesses de braquage/recentrage, simulation au clavier (A/D), recentrage dès que « avance » est coché, sauvegarde JSON, vérification GitHub et mise à jour avec consentement.

Les paramètres restent dans `%APPDATA%\EssieuPilot-Test\profiles.json`. Le programme est installé sous `%LOCALAPPDATA%\Programs\EssieuPilot-Test`, avec un AppId distinct, sans toucher aux jeux et profils existants.

Test de bout en bout : installer `v1.7.0-test.1`, ouvrir « Mises à jour », vérifier puis installer `v1.7.0-test.2`. L'application contrôle l'URL et le SHA-256 publié par GitHub Releases avant l'exécution de l'installateur téléchargé. L'installateur n'est pas signé. Pour tester, accepter l'installation et fermer ATS avant toute intervention sur les plugins (non inclus ici).

**Ce programme simule les essieux uniquement dans sa fenêtre. Aucune DLL ATS, aucune commande physique ni compatibilité Convoi n'est intégrée ou validée.** Il ne doit pas être présenté comme une mise à niveau complète de la version 1.6.1.
