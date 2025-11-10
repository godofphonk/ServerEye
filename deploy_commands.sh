#!/bin/bash
# Команды для переустановки агента на сервере

echo "=== Остановка агента ==="
sudo systemctl stop servereye-agent

echo "=== Замена бинарника ==="
sudo mv /tmp/servereye-agent /usr/local/bin/servereye-agent
sudo chmod +x /usr/local/bin/servereye-agent

echo "=== Очистка старых данных ==="
sudo rm -rf /var/lib/servereye/*

echo "=== Запуск агента ==="
sudo systemctl start servereye-agent

echo "=== Ждем 3 секунды ==="
sleep 3

echo "=== Логи агента (последние 40 строк) ==="
sudo journalctl -u servereye-agent -n 40 --no-pager
