package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

var DNS_IP, SERVER_IP, SERVER_PORT, INTERFACE, SERVER_PUBLIC_KEY_FILE string

func generateClientPrivateKey(clientName string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"wg genkey | tee /etc/wireguard/%s_privatekey",
		clientName,
	))
	clientPrivateKey, err := cmd.Output()
	clientPrivateKey = clientPrivateKey[:len(clientPrivateKey)-1]
	if err != nil {
		log.Fatal(err)
	}
	return string(clientPrivateKey)
}

func generateClientPublicKey(clientPrivateKey, clientName string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"echo %s | wg pubkey | tee /etc/wireguard/%s_publickey",
		clientPrivateKey, clientName,
	))
	clientPublicKey, err := cmd.Output()
	clientPublicKey = clientPublicKey[:len(clientPublicKey)-1]
	if err != nil {
		log.Fatal(err)
	}
	return string(clientPublicKey)
}

func readServerPublicKey() string {
	serverPublicKeyFile, err := os.Open(fmt.Sprintf("/etc/wireguard/%s", SERVER_PUBLIC_KEY_FILE))
	if err != nil {
		log.Fatal(err)
	}
	defer serverPublicKeyFile.Close()

	serverPublicKey, err := io.ReadAll(serverPublicKeyFile)
	serverPublicKey = serverPublicKey[:len(serverPublicKey)-1]
	if err != nil {
		log.Fatal(err)
	}

	return string(serverPublicKey)
}

func readCurrentPeersCount() int {
	wgFileContent, err := os.ReadFile(fmt.Sprintf("/etc/wireguard/%s.conf", INTERFACE))
	if err != nil {
		log.Fatal(err)
	}

	currentPeersCount := strings.Count(string(wgFileContent), "[Peer]")
	return currentPeersCount
}

func createClientConfigurationFile(clientName, clientPrivateKey, serverPublicKey string, currentPeersCount int) {
	clientConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 10.0.0.%d/32
DNS = %s

[Peer]
PublicKey = %s
Endpoint = %s:%s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 20`, clientPrivateKey, currentPeersCount+2, DNS_IP, serverPublicKey, SERVER_IP, SERVER_PORT)

	err := os.WriteFile(fmt.Sprintf("/etc/wireguard/clients/%s.conf", clientName), []byte(clientConf), 0600)
	if err != nil {
		log.Fatal(err)
	}
}

func appendClient(clientPublicKey string, currentPeersCount int) {
	wgFile, err := os.OpenFile(fmt.Sprintf("/etc/wireguard/%s.conf", INTERFACE), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer wgFile.Close()

	wgFile.WriteString(fmt.Sprintf(`

[Peer]
PublicKey = %s
AllowedIPs = 10.0.0.%d/32`, clientPublicKey, currentPeersCount+2))
}

func restartWireguard() {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("systemctl restart wg-quick@%s", INTERFACE))
	_, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
}

func getClientQR(clientName string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("qrencode -t ansiutf8 < /etc/wireguard/clients/%s.conf", clientName))
	qr, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(qr)
}

func init() {
	log.SetFlags(0)

	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found")
	}

	DNS_IP = os.Getenv("DNS_IP")
	SERVER_IP = os.Getenv("SERVER_IP")
	SERVER_PORT = os.Getenv("SERVER_PORT")
	INTERFACE = os.Getenv("INTERFACE")
	SERVER_PUBLIC_KEY_FILE = os.Getenv("SERVER_PUBLIC_KEY_FILE")
}

func eachClient(callback func(client string)) {
	wireguardDir, err := os.Open("/etc/wireguard/")

	if err != nil {
		log.Fatal(err)
	} else {
		files, err := wireguardDir.ReadDir(0)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			parts := strings.Split(file.Name(), "_publickey")
			if len(parts) == 2 {
				callback(parts[0])
			}
		}
	}
}

func main() {
	var clientName string

	if len(os.Args) == 2 {
		clientName = os.Args[1]
	} else {
		fmt.Println("All clients:")
		eachClient(func(client string) {
			fmt.Println(client)
		})
		return
	}

	eachClient(func(client string) {
		if client == clientName {
			fmt.Println(getClientQR(clientName))
			os.Exit(0)
		}
	})

	os.Mkdir("/etc/wireguard/clients", 0600)
	clientPrivateKey := generateClientPrivateKey(clientName)
	clientPublicKey := generateClientPublicKey(clientPrivateKey, clientName)
	serverPublicKey := readServerPublicKey()
	currentPeersCount := readCurrentPeersCount()
	createClientConfigurationFile(clientName, clientPrivateKey, serverPublicKey, currentPeersCount)
	appendClient(clientPublicKey, currentPeersCount)

	fmt.Println("Restarting wireguard...")
	restartWireguard()

	fmt.Println(getClientQR(clientName))
}
