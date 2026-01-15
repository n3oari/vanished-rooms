- Estoy desarrollando una aplicacion para mi trabajo de fin de cursoen Golang llamada vanished-rooms. Esta es una aplicación de mensajeria basada en rooms. Será en CLI mediante la libreria Cobra. Esta enfocada en la privadacid maxima.

- Será puro TCP y la encriptación se realizara mediante intercambio de claves RSA.

- El cliente al inicializar la app entregara el path de su clave privada, e.g -> ./vanished-rooms -u user -p pass -i <path-id-privada -> se obtendra la clave publica y el servidor registrara al usuario.
- Cuando los usuarios entren a una misma room el servidor enviara la clave publica de cada una el resto de usuarios para llevar a cabo la encriptacion.

- La base de datos es será sqlite3 configurada de forma no persistente (en memoria)

- Como herramientas usar: CobraCLI, BubbleTea, Delve, sqlite3. Como IDE uso nvim.

- Esta es la estructura del proyecto actualmente:

tree
├── cmd
│   ├── client.go
│   ├── root.go
│   └── server.go
├── go.mod
├── go.sum
├── internal
│   ├── crypto
│   ├── network
│   │   ├── client.go
│   │   └── server.go
│   └── storage
├── main.go
├── PROMPT.md
├── README.md
├── ui
└── vanished-rooms

- Te iré adjuntando diagramas de secuencia, de clase etc para entender mejor el contexto.

- De ti espero:
  - Opiniones y consejos
  - Que no me generes nada que yo no te pida, el trabajo lo quiero hacer yo.

- Ahora mismo estoy trabajando en la siguiente feature: "añadir feature"
