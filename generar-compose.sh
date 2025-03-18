#!/bin/bash
echo "Nombre del archivo de salida: $1"
echo "Cantidad de clientes: $2"
rm $1
go run scripts/compose_generator.go $1 $2
