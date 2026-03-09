#!/bin/bash
# Попробуем разные способы подключения к серверу
echo "Checking production logs..."

# Способ 1: через docker напрямую (если это локальный продакшен)
if docker ps | grep -q etalon; then
    echo "=== Local production logs ===" 
    docker ps | grep etalon
    exit 0
fi

echo "No local docker containers found"
exit 1
