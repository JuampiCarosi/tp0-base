package main

import (
	"fmt"
	"os"
	"strconv"
	"text/template"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./generar-compose.sh <output_file> <number_of_clients>")
		os.Exit(1)
	}

	outputFileName := os.Args[1]
	outputFile, err := os.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("Error: failed to create output file", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	numClients, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Error: number of clients must be an integer")
		os.Exit(1)
	}

	tmpl, err := template.New(outputFileName).Parse(`name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net

{{range $i := .}}
  client{{$i}}:
    container_name: client{{$i}}
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID={{$i}}
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server
{{end}}

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
`)

	if err != nil {
		fmt.Println("Error: failed to parse template", err)
		os.Exit(1)
	}

	clients := make([]int, numClients)
	for i := range clients {
		clients[i] = i + 1
	}

	err = tmpl.Execute(outputFile, clients)
	if err != nil {
		fmt.Println("Error: failed to execute template", err)
		os.Exit(1)
	}
}
