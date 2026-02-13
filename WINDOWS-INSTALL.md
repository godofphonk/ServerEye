# ServerEye Agent - Windows Installation Guide

## 📦 Что получилось

✅ **Собран .exe файл**: `servereye-agent-windows.exe` (7.3MB)  
✅ **Автоматическая установка**: PowerShell скрипт с генерацией ключа  
✅ **Работает память**: Использует Go runtime memory stats  
✅ **WebSocket подключение**: Полностью функционально  
✅ **Heartbeat**: Отправляется каждые 60 секунд  
✅ **Команды**: ping, status работают  
✅ **Windows Service**: Автозапуск при старте системы  

## 🚀 Автоматическая установка (рекомендуется)

### 1. Скачайте и запустите установщик
Откройте PowerShell **от имени администратора** и выполните:

```powershell
# Скачайте установочный скрипт
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/install-agent-windows.ps1" -OutFile "install-agent-windows.ps1"

# Запустите установку
.\install-agent-windows.ps1
```

### 2. Что делает установщик:
- 📥 **Скачает** последний .exe файл с GitHub releases
- 🔑 **Сгенерирует** уникальный secret key
- 🔄 **Зарегистрирует** ключ на бэкенде API
- ⚙️ **Создаст** конфигурационный файл
- 🔧 **Установит** Windows Service с автозапуском
- 🔥 **Настроит** firewall правила
- 📱 **Покажет** ключ для подключения к Telegram

### 3. Подключитесь к Telegram:
1. Найдите бота: `@ServereyeTG_bot`
2. Отправьте: `/start`
3. Отправьте: `/add YOUR_SECRET_KEY` (ключ покажет установщик)

## 📊 Какие метрики работают

### ✅ **РАБОТАЮТ:**
- **Память**: Go runtime memory stats (Total, Used, Available, %)
- **Uptime**: Время работы агента
- **Heartbeat**: Каждые 60 секунд
- **WebSocket**: Реальное время, переподключение
- **Команды**: ping, status, restart (частично)

### ❌ **НЕ РАБОТАЮТ (пока):**
- CPU температура (требует Windows API)
- Дисковое пространство (требует Windows API)  
- Сетевые интерфейсы (требует Windows API)
- Процессы (требует Windows API)
- Docker (требует Docker Desktop)

## 🔧 Ручная установка (если автоматическая не сработала)

### 1. Скачайте файлы
Скопируйте эти файлы в папку `C:\ServerEye\`:
```
servereye-agent-windows.exe
config-windows.yaml
```

### 2. Отредактируйте конфигурацию
Отредактируйте `config-windows.yaml`:
```yaml
server:
  name: "My-Windows-PC"  # Имя вашего компьютера
  secret_key: "srv_your_unique_key_here"  # Придумайте свой ключ

websocket:
  enabled: true
  url: "wss://servereye-registration-worker.servereye.workers.dev/ws"

metrics:
  memory_usage: true    # Единственная работающая метрика
  interval: "30s"       # Как часто собирать

logging:
  level: "info"
  file: "C:\\ProgramData\\ServerEye\\agent.log"
```

### 3. Запустите агент
```cmd
# Откройте Command Prompt от имени администратора
cd C:\ServerEye
servereye-agent-windows.exe --config config-windows.yaml --log-level debug
```

## 🛠️ Управление сервисом

### Проверить статус:
```powershell
Get-Service "ServerEyeAgent"
```

### Запустить/остановить:
```powershell
Start-Service "ServerEyeAgent"
Stop-Service "ServerEyeAgent"
```

### Просмотр логов:
```powershell
Get-Content "C:\ProgramData\ServerEye\logs\agent.log" -Tail 20
```

## 🗑️ Удаление

### Автоматическое удаление:
```powershell
# Скачайте скрипт удаления
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/godofphonk/ServerEye/master/scripts/uninstall-agent-windows.ps1" -OutFile "uninstall-agent-windows.ps1"

# Запустите удаление
.\uninstall-agent-windows.ps1
```

### Ручное удаление:
```powershell
# Остановите и удалите сервис
Stop-Service "ServerEyeAgent" -Force
& sc.exe delete "ServerEyeAgent"

# Удалите файлы
Remove-Item "C:\Program Files\ServerEye" -Recurse -Force
Remove-Item "C:\ProgramData\ServerEye" -Recurse -Force
```

## 🐛 Troubleshooting

### Ошибки в логах - это НОРМАЛЬНО:
```
WARN: Memory metrics not implemented for windows, returning basic info
WARN: Disk metrics not implemented for windows, returning empty info  
WARN: Network metrics not implemented for windows, returning empty info
WARN: Temperature metrics not implemented for windows, returning zero values
```

### Должно работать:
```
INFO: WebSocket connected successfully
INFO: Heartbeat sent successfully
INFO: Configuration loaded
INFO: Service started successfully
```

### Если не запускается:
1. **Запускайте от имени администратора**
2. **Проверьте статус сервиса**: `Get-Service ServerEyeAgent`
3. **Проверьте логи**: `Get-Content "C:\ProgramData\ServerEye\logs\agent.log" -Tail 50`
4. **Проверьте internet соединение**
5. **Убедитесь что антивирус не блокирует**

## 📈 Пример логов

```
INFO[0000] ServerEye Agent v1.1.0 started successfully  
INFO[0001] Configuration loaded from C:\ProgramData\ServerEye\config.yaml
WARN[0002] Memory metrics not implemented for windows, returning basic info
INFO[0003] WebSocket connected to wss://servereye.../ws
INFO[0060] Heartbeat sent successfully
INFO[0120] Memory metric collected: Total=8GB, Used=2GB, Available=6GB (25%)
```

## 🎯 Что дальше

Для полной функциональности нужно:
1. **Windows API** для CPU, диска, сети
2. **WMI запросы** для температуры  
3. **Event Log** логирование

Но **уже сейчас** агент отправляет память и heartbeat - этого достаточно для базового мониторинга!
