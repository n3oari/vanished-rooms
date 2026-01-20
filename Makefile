ifeq ($(OS),Windows_NT)
    CLEAR = cls
else
    CLEAR = clear
endif

CLIENT_ARGS = -p testtest123415 -k ./resources/privada.pem
DB_PATH = ./internal/storage/sqlite.db

help:
	@echo "Comandos disponibles:"
	@echo "  sv   - Lanza el servidor"
	@echo "  c1       - Cliente user1"
	@echo "  c2       - Cliente user2"
	@echo "  c3       - Cliente user3"
	@echo "  c4       - Cliente user4"
	@echo "  dbreset  - vacia base de datos"
	@echo "  vclean    - limpia consola"


.PHONY: vserver vc1 vc2 vc3 vc4 clear

# --- COMANDOS ---

sv:
	go run . server

c1:
	go run . client -u user1 $(CLIENT_ARGS)

c2:
	go run . client -u user2 $(CLIENT_ARGS)

c3:
	go run . client -u user3 $(CLIENT_ARGS)

c4:
	go run . client -u user4 $(CLIENT_ARGS)

vclean:
	$(CLEAR)

dbreset: 
	sqlite3 $(DB_PATH) "DELETE FROM users; DELETE FROM rooms; DELETE FROM participants;"
	@echo " Base de datos limpiada (users, rooms, participants)."
