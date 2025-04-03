package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/kong"
)

var (
	ImageName string = "lunkan/idapro9ubuntu:latest"
)

// Structure pour les sous-commandes
var CLI struct {
	Help struct{} `cmd:"" default:"1" help:"Affiche l'aide et les informations d'utilisation"`
	List struct{} `cmd:"" help:"Affiche les sessions ida actives"`

	Start struct {
		Name string `name:"name" short:"n" required:"true" help:"Nom de l'instance (obligatoire)"`
	} `cmd:"" help:"Démarre une instance ida dans le répertoire local"`

	Attach struct {
		Name string `name:"name" short:"n" required:"true" help:"Nom de l'instance à laquelle se connecter"`
	} `cmd:"" help:"Se connecte à une instance ida existante"`

	Flush struct {
		Force bool `name:"force" short:"f" help:"Supprime sans demander de confirmation"`
	} `cmd:"" help:"Supprime tous les conteneurs IDA Pro"`
}

func displayHelp() {

	fmt.Println("Il utilise l'image Docker:", ImageName)
	fmt.Println()

	fmt.Println("COMMANDES:")
	fmt.Println("  list            - Affiche toutes les instances IDA Pro en cours d'exécution")
	fmt.Println("  start -n <nom>  - Crée et démarre une nouvelle instance IDA Pro")
	fmt.Println("  attach -n <nom> - Se connecte à une instance existante")
	fmt.Println("  flush [-f]      - Supprime toutes les instances (avec -f pour forcer)")
	fmt.Println()

	fmt.Println("EXEMPLES:")
	fmt.Println("  ida-docker start -n analyse1     - Démarre une session nommée 'analyse1'")
	fmt.Println("  ida-docker attach -n analyse1    - Reprend la session 'analyse1'")
	fmt.Println("  ida-docker list                  - Affiche toutes les sessions")
	fmt.Println("  ida-docker flush                 - Supprime toutes les sessions (avec confirmation)")
	fmt.Println()

	fmt.Println("REMARQUES:")
	fmt.Println("- Le répertoire de travail actuel sera monté dans le conteneur")
	fmt.Println("- Vos analyses et fichiers seront conservés entre les sessions")
	fmt.Println("- L'image Docker sera téléchargée automatiquement si nécessaire")
	fmt.Println()
}

func main() {
	ctx := kong.Parse(&CLI)
	ensureImageExists()
	switch ctx.Command() {
	case "help":
		displayHelp()
	case "list":
		ensureImageExists()
		listIdaSessions()
	case "start":
		ensureImageExists()
		startIda(CLI.Start.Name)
	case "attach":
		ensureImageExists()
		attachToIda(CLI.Attach.Name)
	case "flush":
		flushIdaContainers(CLI.Flush.Force)
	}
}

func listIdaSessions() {
	fmt.Println("Listing des sessions IDA Pro actives...")
	cmd := exec.Command("docker", "ps", "--filter", "ancestor=idapro9ubuntu")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func startIda(name string) {
	fmt.Printf("Démarrage d'une nouvelle instance IDA Pro nommée '%s'...\n", name)

	// Vérifier si un conteneur avec ce nom existe déjà
	checkCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err == nil && len(output) > 0 {
		fmt.Printf("Un conteneur nommé '%s' existe déjà.\n", name)
		fmt.Println("Vous pouvez vous y connecter avec 'attach' ou choisir un autre nom.")
		os.Exit(1)
	}

	// Lancement d'une nouvelle instance
	newIdaWithName(name)
}

func attachToIda(name string) {
	fmt.Printf("Connexion à l'instance IDA Pro '%s'...\n", name)

	// Vérifier si le conteneur existe et est en cours d'exécution
	checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil || len(output) == 0 {
		fmt.Printf("Aucun conteneur en cours d'exécution nommé '%s' n'a été trouvé.\n", name)

		// Vérifier si le conteneur existe mais est arrêté
		checkStoppedCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
		stoppedOutput, stoppedErr := checkStoppedCmd.Output()
		if stoppedErr == nil && len(stoppedOutput) > 0 {
			fmt.Println("Le conteneur existe mais n'est pas en cours d'exécution.")
			fmt.Printf("Démarrage du conteneur '%s'...\n", name)

			startCmd := exec.Command("docker", "start", name)
			startCmd.Stdout = os.Stdout
			startCmd.Stderr = os.Stderr
			err = startCmd.Run()
			if err != nil {
				fmt.Printf("Erreur lors du démarrage du conteneur: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Utilisez 'start' pour créer un nouveau conteneur.")
			os.Exit(1)
		}
	}

	// Configuration des droits d'accès X11
	setupX11()

	// Exécuter IDA Pro dans le conteneur
	dockerExec := exec.Command(
		"docker", "exec", "-it",
		name,
		"/opt/ida-pro-9.0/ida",
	)

	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr

	fmt.Println("Lancement d'IDA Pro...")
	err = dockerExec.Run()
	if err != nil {
		fmt.Printf("Erreur lors de l'exécution d'IDA Pro: %v\n", err)
		os.Exit(1)
	}
}

func flushIdaContainers(force bool) {
	// Récupérer la liste des conteneurs IDA Pro
	cmd := exec.Command("docker", "ps", "-a", "--filter", "ancestor=idapro9ubuntu", "--format", "{{.Names}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Erreur lors de la recherche des conteneurs: %v\n", err)
		os.Exit(1)
	}

	// Lire les noms des conteneurs
	containerNames := strings.Split(strings.TrimSpace(out.String()), "\n")
	if containerNames[0] == "" && len(containerNames) == 1 {
		fmt.Println("Aucun conteneur IDA Pro trouvé.")
		return
	}

	// Afficher le nombre de conteneurs trouvés
	fmt.Printf("Trouvé %d conteneur(s) IDA Pro.\n", len(containerNames))

	// Demander confirmation si force est false
	if !force {
		fmt.Print("Êtes-vous sûr de vouloir supprimer tous ces conteneurs? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Opération annulée.")
			return
		}
	}

	// Arrêter et supprimer chaque conteneur
	for _, name := range containerNames {
		if name == "" {
			continue
		}
		fmt.Printf("Suppression du conteneur: %s\n", name)
		rmCmd := exec.Command("docker", "rm", "-f", name)
		err := rmCmd.Run()
		if err != nil {
			fmt.Printf("Erreur lors de la suppression du conteneur %s: %v\n", name, err)
		}
	}

	fmt.Println("Tous les conteneurs IDA Pro ont été supprimés.")
}

func ensureImageExists() {
	fmt.Printf("Vérification de l'image %s...\n", ImageName)

	cmd := exec.Command("docker", "image", "inspect", ImageName)
	err := cmd.Run()

	// Image non trouvée
	if err != nil {
		fmt.Printf("Image %s non trouvée localement.\n", ImageName)
		fmt.Printf("Téléchargement automatique de l'image %s...\n", ImageName)
		pullImage()
	}
}

func pullImage() {
	fmt.Printf("Téléchargement/mise à jour de l'image %s...\n", ImageName)

	cmd := exec.Command("docker", "pull", ImageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Erreur lors du téléchargement de l'image: %v\n", err)
		fmt.Println()
		fmt.Println("Si l'image est hébergée sur un registre privé, authentifiez-vous d'abord avec:")
		fmt.Println("  docker login [registry-url]")
		fmt.Println()
		fmt.Println("Alternativement, vous pouvez construire l'image localement avec le Dockerfile.")
		os.Exit(1)
	}

	fmt.Printf("Image %s prête à l'utilisation.\n", ImageName)
}

// Version modifiée de NewIda qui accepte un nom
func newIdaWithName(name string) {

	if os.Geteuid() != 0 {
		fmt.Println("Ce programme nécessite des privilèges administrateur (sudo).")
		fmt.Println("Veuillez relancer avec sudo.")
		os.Exit(1)
	}

	// Configuration X11
	setupX11()

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Erreur lors de la récupération du répertoire de travail: %v\n", err)
		os.Exit(1)
	}

	display := os.Getenv("DISPLAY")

	dockerCmd := exec.Command(
		"docker", "run", "-itd",
		"--name", name, // Utilisation du nom fourni
		"--env", fmt.Sprintf("DISPLAY=%s", display),
		"--volume", "/tmp/.X11-unix:/tmp/.X11-unix",
		"--volume", fmt.Sprintf("%s:/home/ubuntu", pwd),
		"idapro9ubuntu",
	)

	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	fmt.Println("Création du conteneur Docker...")
	err = dockerCmd.Run()
	if err != nil {
		fmt.Printf("Erreur lors de la création du conteneur: %v\n", err)
		os.Exit(1)
	}

	// Lancement automatique d'IDA Pro après création du conteneur
	dockerExec := exec.Command(
		"docker", "exec", "-itd",
		name,
		"/opt/ida-pro-9.0/ida",
	)

	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr

	fmt.Println("Lancement d'IDA Pro...")
	err = dockerExec.Run()
	if err != nil {
		fmt.Printf("Erreur lors de l'exécution d'IDA Pro: %v\n", err)
		os.Exit(1)
	}
}

// Fonction commune pour configurer X11
func setupX11() {
	display := os.Getenv("DISPLAY")
	if display == "" {
		fmt.Println("Variable DISPLAY non définie. Impossible d'accéder à X11.")
		os.Exit(1)
	}

	fmt.Println("Configuration des droits d'accès X11...")
	xhostCmd := exec.Command("xhost", "+local:docker")
	xhostCmd.Stdout = os.Stdout
	xhostCmd.Stderr = os.Stderr
	err := xhostCmd.Run()
	if err != nil {
		fmt.Printf("Avertissement: Impossible de configurer xhost: %v\n", err)
		fmt.Println("L'affichage X11 pourrait ne pas fonctionner correctement.")
	}
}
