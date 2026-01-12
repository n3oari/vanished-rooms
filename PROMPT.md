- Voy a hacer mi trabajo de fin de curso que consta de lo siguiente:


- App en golang CLI: será una chat room enfocada en la privacidad maxima via TCP. Tendrá una base de datos sqlite3 no 
persistente, es decir, se ejecutrara en RAM y se eliminará.

- La encriptacón se hara mediante claves publicas y privadas de cada usuario

- Herramientas / librerias:  onion(tor), crypto, CobraCLI, Bubble tea, Testify, net ... etc

    
Por ahora la estructura del proyecto será asi:

/vanished-rooms
├── cmd/                # Entry points de Cobra
│   ├── root.go
│   ├── client.go       # Comando para iniciar el cliente
│   └── server.go       # Comando para iniciar el servidor/retransmitidor
├── internal/           # Código privado que no quieres que otros importen
│   ├── crypto/         # RSA + AES (Tu lógica de privacidad)
│   ├── storage/        # Lógica de SQLite
│   ├── network/        # Protocolo de comunicación y Tor
|   ├── ui/             # Estetica cmd , Bubble Tea  UI
├── go.mod
└── main.go             # Solo llama a cmd.Execute()

- Te adjunto foto del diagrama de secuencia (es un boceto)

- De ti espero:
    - Opininones y consejos
    - Que no me generes nada que yo no te pida, el trabajo lo quiero hacer yo.

- Ahora mismo quiero centrarme en la 1º feature:

/vanished-rooms
├── cmd/                # Entry points de Cobra
│   ├── root.go
│   ├── client.go       # Comando para iniciar el cliente
│   └── server.go       # Comando para iniciar el servidor/retransmitidor

> - **Servidor:** Crear un servidor TCP simple en Go que reciba un mensaje y lo reenvíe a todos los conectados.
    
>- **Cliente:**  que se conecte a la IP del servidor y permita escribir en la terminal.

> - Implementar **Cobra**
    
> - **Resultado:** Un chat funcional, pero totalmente inseguro (texto plano)


