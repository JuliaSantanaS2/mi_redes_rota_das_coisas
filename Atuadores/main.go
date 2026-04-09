package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// Função para buscar variaveis de ambiente definidas no Docker
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {

	nome := getEnv("ATUADOR_NOME", "Atuador Generico")
	porta := getEnv("ATUADOR_PORTA", "9090")

	endereco := ":" + porta

	fmt.Printf("Inicializando %s...\n", nome)
	iniciarAtuador(nome, endereco)
}

func iniciarAtuador(nome string, porta string) {
	ln, err := net.Listen("tcp", porta)
	if err != nil {
		fmt.Printf("[ERRO] Atuador '%s' na porta %s: %v\n", nome, porta, err)
		return
	}
	defer ln.Close()
	fmt.Printf("[OK] Atuador '%s' pronto aguardando comandos na porta %s\n", nome, porta)

	for {
		conn, err := ln.Accept()
		if err == nil {
			go processarComando(conn, nome)
		}
	}
}

func processarComando(conn net.Conn, nome string) {
	defer conn.Close()

	mensagem, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return
	}
	comando := strings.TrimSpace(mensagem)

	// Recebendo e Mandando a Mesangem
	fmt.Printf("\n>>> [%s] Recebeu a ordem: %s <<<\n", nome, comando)
	fmt.Fprintf(conn, "CONFIRMADO: %s\n", comando)
}
