COMMANDES:
  list            - Affiche toutes les instances IDA Pro en cours d'exécution
  start -n <nom>  - Crée et démarre une nouvelle instance IDA Pro
  attach -n <nom> - Se connecte à une instance existante
  flush [-f]      - Supprime toutes les instances (avec -f pour forcer)

EXEMPLES:
  ida-docker start -n analyse1     - Démarre une session nommée 'analyse1'
  ida-docker attach -n analyse1    - Reprend la session 'analyse1'
  ida-docker list                  - Affiche toutes les sessions
  ida-docker flush                 - Supprime toutes les sessions (avec confirmation)

REMARQUES:
- Le répertoire de travail actuel sera monté dans le conteneur
- Vos analyses et fichiers seront conservés entre les sessions
- L'image Docker sera téléchargée automatiquement si nécessaire
