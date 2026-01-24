package cmd

import (
	"crypto/rsa"
	"log"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/ui" // Importamos el controlador de la interfaz

	"github.com/spf13/cobra"
)

var (
	// Variables para almacenar los valores de las Flags
	hostAddr  string
	proxyAddr string
	tor       bool
)

// clientCmd define el comando para lanzar el cliente de Vanished Rooms
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Ejecuta el cliente con interfaz visual integrada",
	Long:  `Lanza la terminal en modo oscuro (negro y rojo) para conectar con el servidor de Vanished Rooms.`,
	Run: func(cmd *cobra.Command, args []string) {
		// LLAMADA CLAVE:
		// No ejecutamos la red aquí, sino que lanzamos la UI profesional.
		// StartApp se encargará de inicializar tview, poner el fondo negro
		// y gestionar el flujo de Login -> Red -> Chat.
		ui.StartApp(hostAddr, tor, proxyAddr)
	},
}

func init() {
	// 1. Registramos el comando en la raíz (rootCmd)
	rootCmd.AddCommand(clientCmd)

	// 2. Definición de Flags de configuración de red
	// Estas banderas permiten al usuario configurar la conexión desde la terminal.
	clientCmd.Flags().StringVarP(&hostAddr, "host", "H", "localhost:8080", "Dirección del servidor (.onion o IP)")
	clientCmd.Flags().BoolVarP(&tor, "tor", "t", false, "Activar modo TOR (SOCKS5 127.0.0.1:9050)")
	clientCmd.Flags().StringVarP(&proxyAddr, "proxy", "x", "", "Dirección del Proxy HTTP opcional")

	// Nota: No marcamos campos como obligatorios aquí porque el Formulario de Login
	// capturará el resto de los datos (User, Pass, Key) visualmente.
}

func prepareKeys(path string) (*rsa.PrivateKey, string) {
	privKey, err := cryptoutils.LoadPrivateKey(path)
	if err != nil {
		log.Fatalf("Error loading private key: %v", err)
	}

	pubKeyToSend, err := cryptoutils.EncodePublicKeyToBase64(privKey)
	if err != nil {
		log.Fatalf("Error generating a public key: %v", err)
	}

	return privKey, pubKeyToSend
}
