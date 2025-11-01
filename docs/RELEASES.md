# 📦 Release Process

## Как создать новый релиз

### 1. Убедитесь что все изменения закоммичены

```bash
git status
git add .
git commit -m "chore: prepare for release v1.0.0"
git push origin master
```

### 2. Создайте и запуште тег

```bash
# Создаем тег (версия должна начинаться с 'v')
git tag -a v1.0.0 -m "Release v1.0.0"

# Пушим тег в GitHub
git push origin v1.0.0
```

### 3. GitHub Actions автоматически:

1. ✅ Соберет бинарники для всех платформ
2. ✅ Создаст GitHub Release
3. ✅ Загрузит файлы в релиз
4. ✅ Сгенерирует checksums

### 4. Проверьте релиз

Перейдите на: https://github.com/godofphonk/ServerEye/releases

## Что будет в релизе

### Bot бинарники:
- `servereye-bot-linux-amd64` - Linux (Intel/AMD)
- `servereye-bot-darwin-amd64` - macOS (Intel)
- `servereye-bot-darwin-arm64` - macOS (M1/M2)
- `servereye-bot-windows-amd64.exe` - Windows

### Agent бинарники:
- `servereye-agent-linux-amd64` - Linux (Intel/AMD)
- `servereye-agent-linux-arm64` - Linux (ARM) - для Raspberry Pi, AWS Graviton
- `servereye-agent-darwin-amd64` - macOS (Intel)
- `servereye-agent-darwin-arm64` - macOS (M1/M2)
- `servereye-agent-windows-amd64.exe` - Windows

### Дополнительно:
- `checksums.txt` - SHA256 хэши для проверки

## Semantic Versioning

Используем [SemVer](https://semver.org/):

- **v1.0.0** - Major release (breaking changes)
- **v1.1.0** - Minor release (new features, backwards compatible)
- **v1.0.1** - Patch release (bug fixes)

## Примеры

### Первый релиз
```bash
git tag -a v1.0.0 -m "Initial release"
git push origin v1.0.0
```

### Фича релиз
```bash
git tag -a v1.1.0 -m "Add Docker management features"
git push origin v1.1.0
```

### Bugfix релиз
```bash
git tag -a v1.0.1 -m "Fix memory leak in Redis client"
git push origin v1.0.1
```

## Удаление тега (если ошиблись)

```bash
# Локально
git tag -d v1.0.0

# На GitHub
git push --delete origin v1.0.0
```

## Проверка билда локально

Перед созданием релиза убедитесь что все компилируется:

```bash
# Bot
GOOS=linux GOARCH=amd64 go build ./cmd/bot
GOOS=darwin GOARCH=amd64 go build ./cmd/bot
GOOS=windows GOARCH=amd64 go build ./cmd/bot

# Agent
GOOS=linux GOARCH=amd64 go build ./cmd/agent
GOOS=linux GOARCH=arm64 go build ./cmd/agent
```
