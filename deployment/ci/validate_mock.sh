#!/bin/bash

# Lista todas as interfaces do projeto (nomes únicos)
interfaces=$(grep -rhoE 'type[[:space:]]+[A-Za-z0-9_]+[[:space:]]+interface[[:space:]]*\{' . --include="*.go" | sed -E 's/type[[:space:]]+([A-Za-z0-9_]+)[[:space:]]+interface.*/\1/' | sort | uniq)

echo "Interfaces encontradas:"
echo "$interfaces"
echo

missing=0

# Para cada interface, verifica se existe algum arquivo _mock.go que a referencia
for iface in $interfaces; do
    if grep -r -w "Mock$iface" . --include="*mock_*.go" > /dev/null; then
        echo "✅ Mock encontrado para interface: Mock$iface"
    else
        echo "❌ Mock NÃO encontrado para interface: Mock$iface"
        missing=$((missing+1))
    fi
done

if [ $missing -eq 0 ]; then
    echo "\nTodas as interfaces possuem mocks gerados."
    exit 0
else
    echo "\n$missing interface(s) sem mocks."
    exit 1
fi