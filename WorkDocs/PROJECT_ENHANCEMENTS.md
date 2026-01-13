# ServerEye - Project Enhancement Roadmap

## Overview

This document outlines strategic improvements for the ServerEye project focusing on architectural patterns, code quality, and enterprise-level features. The enhancements are designed to improve maintainability, testability, and scalability while following modern Go development practices.

## Architectural Improvements

### 1. Dependency Injection with Wire ready

#### Текущее состояние
- Manual dependency initialization in `agent.go`
- Hard-coded dependencies
- Difficult unit testing
- Tight coupling between components

#### Предлагаемое решение: Google Wire
```go
// internal/agent/wire.go
//go:build wireinject
// +build wireinject

package agent

import (
    "github.com/google/wire"
    "github.com/godofphonk/ServerEye/internal/config"
    "github.com/godofphonk/ServerEye/pkg/metrics"
    "github.com/godofphonk/ServerEye/pkg/commands"
    "github.com/godofphonk/ServerEye/pkg/docker"
    "github.com/godofphonk/ServerEye/pkg/websocket"
)

// ProviderSet is a collection of providers for the agent package
var ProviderSet = wire.NewSet(
    config.NewConfigLoader,
    metrics.NewCPUMetrics,
    metrics.NewSystemMonitor,
    docker.NewClient,
    websocket.NewClient,
    commands.NewWebSocketCommandConsumer,
    metrics.NewWebSocketPublisher,
    NewAgent,
)

// InitializeAgent creates a new agent with all dependencies
func InitializeAgent(configPath string, logLevel string) (*Agent, error) {
    wire.Build(ProviderSet)
    return nil, nil
}
```

#### Преимущества
- **Тестируемость**: Легкая инъекция моков
- **Гибкость**: Конфигурация зависимостей во время выполнения
- **Поддерживаемость**: Четкий граф зависимостей
- **Разделение ответственности**: Интерфейсно-ориентированный дизайн

#### Этапы реализации
1. Определить интерфейсы для всех основных компонентов
2. Создать наборы провайдеров Wire
3. Рефакторить `agent.go` для использования внедренных зависимостей
4. Обновить unit тесты с mock зависимостями
5. Добавить опции конфигурации зависимостей

### 2. Interface-Driven Design

#### Текущие проблемы
- Использование конкретных типов
- Сложность создания моков для тестов
- Плотная связанность с реализациями

#### Предлагаемая иерархия интерфейсов
```go
// pkg/interfaces/metrics.go
package interfaces

import "context"

type MetricsCollector interface {
    CollectCPU(ctx context.Context) (*CPUMetrics, error)
    CollectMemory(ctx context.Context) (*MemoryMetrics, error)
    CollectDisk(ctx context.Context) (*DiskMetrics, error)
    Start(ctx context.Context) error
    Stop() error
}

type MetricsPublisher interface {
    Publish(ctx context.Context, metric interface{}) error
    PublishBatch(ctx context.Context, metrics []interface{}) error
    IsConnected() bool
}

// pkg/interfaces/commands.go
package interfaces

type CommandHandler interface {
    HandleCommand(ctx context.Context, cmd *protocol.Message) (*protocol.Message, error)
}

type CommandConsumer interface {
    Start(ctx context.Context) error
    Stop() error
    RegisterHandler(handler CommandHandler) error
}

// pkg/interfaces/docker.go
package interfaces

type DockerManager interface {
    ListContainers(ctx context.Context) ([]ContainerInfo, error)
    StartContainer(ctx context.Context, id string) error
    StopContainer(ctx context.Context, id string) error
    RestartContainer(ctx context.Context, id string) error
    GetContainerStats(ctx context.Context, id string) (*ContainerStats, error)
}
```

#### План реализации
1. Define core interfaces for all major components
2. Refactor existing implementations to satisfy interfaces
3. Update agent to use interface types
4. Create mock implementations for testing
5. Add interface compliance tests

### 3. Улучшение системы конфигурации

#### Текущие ограничения
- Single configuration file
- No environment-specific configs
- Limited validation
- No hot-reload capability

#### Предлагаемая система конфигурации
```go
// internal/config/manager.go
package config

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type Manager interface {
    Load(ctx context.Context) (*AgentConfig, error)
    Reload(ctx context.Context) error
    Watch(ctx context.Context) (<-chan *AgentConfig, error)
    Validate(cfg *AgentConfig) error
}

type manager struct {
    configPath    string
    currentConfig *AgentConfig
    mu            sync.RWMutex
    watchers      []chan *AgentConfig
}

func (m *manager) Watch(ctx context.Context) (<-chan *AgentConfig, error) {
    ch := make(chan *AgentConfig, 1)
    m.mu.Lock()
    m.watchers = append(m.watchers, ch)
    m.mu.Unlock()
    
    go m.watchFile(ctx, ch)
    return ch, nil
}

// internal/config/loader.go
type Loader interface {
    LoadFromPath(path string) (*AgentConfig, error)
    LoadFromEnv() (*AgentConfig, error)
    Merge(base, override *AgentConfig) (*AgentConfig, error)
}

type loader struct {
    validators []Validator
}

func (l *loader) LoadFromPath(path string) (*AgentConfig, error) {
    // Support multiple formats: YAML, JSON, TOML
    ext := filepath.Ext(path)
    switch ext {
    case ".yaml", ".yml":
        return l.loadYAML(path)
    case ".json":
        return l.loadJSON(path)
    case ".toml":
        return l.loadTOML(path)
    default:
        return nil, fmt.Errorf("unsupported config format: %s", ext)
    }
}
```

#### Возможности
- **Multiple formats**: YAML, JSON, TOML support
- **Environment override**: Environment variable overrides
- **Hot-reload**: File watching with automatic reload
- **Validation**: Comprehensive configuration validation
- **Profiles**: Environment-specific configurations

### 4. Расширенное управление ошибками

#### Текущее состояние
- Basic error returns
- Limited error context
- No error classification
- Minimal error recovery

#### Предлагаемая система обработки ошибок
```go
// pkg/errors/types.go
package errors

import (
    "fmt"
    "runtime"
    "time"
)

type Type string

const (
    TypeValidation    Type = "validation"
    TypeNetwork       Type = "network"
    TypeSystem        Type = "system"
    TypeConfig        Type = "config"
    TypePermission    Type = "permission"
    TypeTimeout       Type = "timeout"
    TypeInternal      Type = "internal"
)

type Severity string

const (
    SeverityLow      Severity = "low"
    SeverityMedium   Severity = "medium"
    SeverityHigh     Severity = "high"
    SeverityCritical Severity = "critical"
)

type Error struct {
    Type        Type              `json:"type"`
    Severity    Severity          `json:"severity"`
    Code        string            `json:"code"`
    Message     string            `json:"message"`
    Cause       error             `json:"cause,omitempty"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Stack       []string          `json:"stack,omitempty"`
    Timestamp   time.Time         `json:"timestamp"`
    Retryable   bool              `json:"retryable"`
}

func (e *Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(errType Type, code, message string) *Error {
    return &Error{
        Type:      errType,
        Code:      code,
        Message:   message,
        Timestamp: time.Now(),
        Stack:     captureStack(),
    }
}

func (e *Error) WithCause(cause error) *Error {
    e.Cause = cause
    return e
}

func (e *Error) WithContext(key string, value interface{}) *Error {
    if e.Context == nil {
        e.Context = make(map[string]interface{})
    }
    e.Context[key] = value
    return e
}

func (e *Error) WithRetryable() *Error {
    e.Retryable = true
    return e
}
```

#### Error Handling Strategy
```go
// pkg/errors/handler.go
type Handler interface {
    Handle(ctx context.Context, err error) error
    ShouldRetry(err error) bool
    GetRetryDelay(attempt int, err error) time.Duration
}

type handler struct {
    logger Logger
    metrics Metrics
}

func (h *handler) Handle(ctx context.Context, err error) error {
    var serverErr *Error
    if !errors.As(err, &serverErr) {
        // Wrap unknown errors
        serverErr = New(TypeInternal, "UNKNOWN_ERROR", err.Error()).WithCause(err)
    }

    // Log with appropriate level
    h.logError(serverErr)

    // Update metrics
    h.metrics.IncrementError(serverErr.Type, serverErr.Severity)

    // Decide on recovery action
    if h.ShouldRetry(serverErr) {
        return serverErr // Return for retry logic
    }

    return nil // Don't retry
}
```

### 5. Улучшение наблюдаемости

#### Текущее состояние
- Basic logging with logrus
- No structured metrics
- No distributed tracing
- Limited monitoring capabilities

#### Предлагаемая стек наблюдаемости

##### Structured Logging
```go
// pkg/logging/logger.go
package logging

import (
    "context"
    "os"
    
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)
    With(fields ...Field) Logger
    WithContext(ctx context.Context) Logger
}

type Field struct {
    Key   string
    Value interface{}
}

type logger struct {
    zap *zap.Logger
}

func NewLogger(config Config) (Logger, error) {
    zapConfig := zap.NewProductionConfig()
    zapConfig.Level = zap.NewAtomicLevelAt(toZapLevel(config.Level))
    zapConfig.OutputPaths = config.OutputPaths
    zapConfig.ErrorOutputPaths = config.ErrorOutputPaths
    
    if config.Development {
        zapConfig.Development = true
        zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
    }
    
    zapLogger, err := zapConfig.Build()
    if err != nil {
        return nil, err
    }
    
    return &logger{zap: zapLogger}, nil
}
```

##### Metrics Collection
```go
// pkg/metrics/collector.go
package metrics

import (
    "context"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

type Collector interface {
    RecordCounter(name string, tags map[string]string, value float64)
    RecordGauge(name string, tags map[string]string, value float64)
    RecordHistogram(name string, tags map[string]string, value float64)
    RecordTimer(name string, tags map[string]string, duration time.Duration)
}

type collector struct {
    meter metric.Meter
    
    counters map[string]metric.Float64Counter
    gauges   map[string]metric.Float64ObservableGauge
    histograms map[string]metric.Float64Histogram
}

func NewCollector() (Collector, error) {
    meter := otel.Meter("servereye-agent")
    
    return &collector{
        meter:      meter,
        counters:   make(map[string]metric.Float64Counter),
        gauges:     make(map[string]metric.Float64ObservableGauge),
        histograms: make(map[string]metric.Float64Histogram),
    }, nil
}
```

##### Distributed Tracing
```go
// pkg/tracing/tracer.go
package tracing

import (
    "context"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

type Tracer interface {
    StartSpan(ctx context.Context, name string) (context.Context, Span)
    SpanFromContext(ctx context.Context) Span
}

type Span interface {
    SetAttributes(attributes ...Attribute)
    SetStatus(code trace.Status)
    End()
    AddEvent(name string, attributes ...Attribute)
}

type tracer struct {
    tracer trace.Tracer
}

func NewTracer(serviceName string) (Tracer, error) {
    // Initialize OpenTelemetry tracer
    tracer := otel.Tracer(serviceName)
    return &tracer{tracer: tracer}, nil
}
```

### 6. Улучшение инфраструктуры тестирования

#### Текущее состояние
- Basic unit tests
- Limited integration tests
- No end-to-end tests
- No test utilities

#### Предлагаемая стратегия тестирования

##### Test Framework
```go
// pkg/testing/testsuite.go
package testing

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/suite"
    "github.com/golang/mock/gomock"
)

type TestSuite struct {
    suite.Suite
    Ctx        context.Context
    Cancel     context.CancelFunc
    Ctrl       *gomock.Controller
    StartTime  time.Time
}

func (s *TestSuite) SetupTest() {
    s.Ctx, s.Cancel = context.WithTimeout(context.Background(), 30*time.Second)
    s.Ctrl = gomock.NewController(s.T())
    s.StartTime = time.Now()
}

func (s *TestSuite) TearDownTest() {
    s.Cancel()
    s.Ctrl.Finish()
}

// Integration test suite
type IntegrationTestSuite struct {
    TestSuite
    DockerCompose *DockerCompose
    TestConfig    *Config
}

func (s *IntegrationTestSuite) SetupSuite() {
    // Start test infrastructure
    s.DockerCompose = NewDockerCompose("testdata/docker-compose.yml")
    s.Require().NoError(s.DockerCompose.Up())
    
    // Wait for services to be ready
    s.Require().NoError(s.DockerCompose.WaitForServices(60*time.Second))
}

func (s *IntegrationTestSuite) TearDownSuite() {
    if s.DockerCompose != nil {
        s.DockerCompose.Down()
    }
}
```

##### Mock Generation
```go
// pkg/mocks/generate.go
//go:generate mockgen -source=pkg/interfaces/metrics.go -destination=pkg/mocks/metrics.go
//go:generate mockgen -source=pkg/interfaces/commands.go -destination=pkg/mocks/commands.go
//go:generate mockgen -source=pkg/interfaces/docker.go -destination=pkg/mocks/docker.go
//go:generate mockgen -source=pkg/websocket/client.go -destination=pkg/mocks/websocket.go

package mocks

import (
    "context"
    "github.com/godofphonk/ServerEye/pkg/interfaces"
    "github.com/golang/mock/gomock"
)

// Mock implementations will be generated here
```

##### Test Utilities
```go
// pkg/testing/utils.go
package testing

import (
    "context"
    "fmt"
    "net"
    "time"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type TestContainer struct {
    testcontainers.Container
    Host string
    Port int
}

func StartTestContainer(ctx context.Context, image string) (*TestContainer, error) {
    req := testcontainers.ContainerRequest{
        Image:        image,
        ExposedPorts: []string{"8080/tcp"},
        WaitingFor:   wait.ForHTTP("/health"),
    }
    
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        return nil, err
    }
    
    host, err := container.Host(ctx)
    if err != nil {
        return nil, err
    }
    
    port, err := container.MappedPort(ctx, "8080")
    if err != nil {
        return nil, err
    }
    
    return &TestContainer{
        Container: container,
        Host:      host,
        Port:      port.Int(),
    }, nil
}

func GetFreePort() (int, error) {
    addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
    if err != nil {
        return 0, err
    }
    
    l, err := net.ListenTCP("tcp", addr)
    if err != nil {
        return 0, err
    }
    defer l.Close()
    
    return l.Addr().(*net.TCPAddr).Port, nil
}
```

### 7. Оптимизация производительности

#### Текущие проблемы с производительностью
- Synchronous operations blocking main loop
- No connection pooling
- Inefficient memory usage
- No caching mechanisms

#### Предлагаемые оптимизации

##### Connection Pooling
```go
// pkg/pool/websocket.go
package pool

import (
    "context"
    "sync"
    
    "github.com/godofphonk/ServerEye/pkg/websocket"
)

type WebSocketPool interface {
    Get(ctx context.Context) (websocket.Client, error)
    Put(client websocket.Client) error
    Close() error
    Stats() PoolStats
}

type webSocketPool struct {
    factory    func() (websocket.Client, error)
    pool       chan websocket.Client
    mu         sync.RWMutex
    maxSize    int
    currentSize int
    stats      PoolStats
}

func (p *webSocketPool) Get(ctx context.Context) (websocket.Client, error) {
    select {
    case client := <-p.pool:
        p.mu.Lock()
        p.currentSize--
        p.mu.Unlock()
        return client, nil
    default:
        // Create new connection if pool is empty
        return p.factory()
    }
}
```

##### Caching Layer
```go
// pkg/cache/cache.go
package cache

import (
    "context"
    "time"
    
    "github.com/patrickmn/go-cache"
)

type Cache interface {
    Get(ctx context.Context, key string) (interface{}, bool)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
}

type memoryCache struct {
    cache *cache.Cache
}

func NewMemoryCache(defaultExpiration time.Duration) Cache {
    return &memoryCache{
        cache: cache.New(defaultExpiration, 10*time.Minute),
    }
}

// Redis cache implementation for distributed scenarios
type redisCache struct {
    client redis.Client
    prefix string
}
```

##### Async Processing
```go
// pkg/async/processor.go
package async

import (
    "context"
    "sync"
    
    "github.com/robfig/cron/v3"
)

type Processor interface {
    Submit(ctx context.Context, task Task) error
    SubmitBatch(ctx context.Context, tasks []Task) error
    Start(ctx context.Context) error
    Stop() error
}

type Task interface {
    Execute(ctx context.Context) error
    ID() string
    Priority() int
}

type processor struct {
    workers   int
    taskQueue chan Task
    workersWg sync.WaitGroup
    cron      *cron.Cron
}

func (p *processor) worker(ctx context.Context) {
    for {
        select {
        case task := <-p.taskQueue:
            if err := task.Execute(ctx); err != nil {
                // Handle task error
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### 8. Улучшения безопасности

#### Текущие меры безопасности
- Basic non-root user
- Simple API key authentication
- Limited input validation

#### Предлагаемые улучшения безопасности

##### Authentication & Authorization
```go
// pkg/security/auth.go
package security

import (
    "context"
    "time"
    
    "github.com/golang-jwt/jwt/v4"
)

type AuthManager interface {
    GenerateToken(ctx context.Context, claims Claims) (string, error)
    ValidateToken(ctx context.Context, token string) (*Claims, error)
    RefreshToken(ctx context.Context, token string) (string, error)
}

type Claims struct {
    ServerID   string    `json:"server_id"`
    Permissions []string `json:"permissions"`
    ExpiresAt  time.Time `json:"exp"`
    IssuedAt   time.Time `json:"iat"`
}

type authManager struct {
    secretKey []byte
    issuer    string
}

func (a *authManager) GenerateToken(ctx context.Context, claims Claims) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(a.secretKey)
}
```

##### Input Validation
```go
// pkg/security/validator.go
package security

import (
    "context"
    "regexp"
    
    "github.com/go-playground/validator/v10"
)

type Validator interface {
    ValidateStruct(ctx context.Context, s interface{}) error
    ValidateField(ctx context.Context, field interface{}, rule string) error
    SanitizeInput(ctx context.Context, input string) string
}

type validator struct {
    validate *validator.Validate
}

func NewValidator() Validator {
    v := validator.New()
    
    // Register custom validators
    v.RegisterValidation("server_key", validateServerKey)
    v.RegisterValidation("container_id", validateContainerID)
    
    return &validator{validate: v}
}

func validateServerKey(fl validator.FieldLevel) bool {
    serverKey := fl.Field().String()
    matched, _ := regexp.MatchString(`^srv_[a-f0-9]{32}$`, serverKey)
    return matched
}
```

##### Rate Limiting
```go
// pkg/security/ratelimit.go
package security

import (
    "context"
    "time"
    
    "golang.org/x/time/rate"
)

type RateLimiter interface {
    Allow(ctx context.Context, key string) bool
    Wait(ctx context.Context, key string) error
    SetLimit(key string, limit rate.Limit)
}

type rateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    defaultLimit rate.Limit
}

func (r *rateLimiter) Allow(ctx context.Context, key string) bool {
    limiter := r.getLimiter(key)
    return limiter.Allow()
}

func (r *rateLimiter) getLimiter(key string) *rate.Limiter {
    r.mu.RLock()
    limiter, exists := r.limiters[key]
    r.mu.RUnlock()
    
    if !exists {
        r.mu.Lock()
        defer r.mu.Unlock()
        
        // Double-check after acquiring write lock
        if limiter, exists = r.limiters[key]; !exists {
            limiter = rate.NewLimiter(r.defaultLimit, 1)
            r.limiters[key] = limiter
        }
    }
    
    return limiter
}
```

## План реализации

### Phase 1: Foundation (Weeks 1-2)
1. **Определение интерфейсов** - Определить основные интерфейсы
2. **Интеграция Wire** - Настроить внедрение зависимостей
3. **Базовый фреймворк тестирования** - Тестовые утилиты и моки
4. **Улучшение конфигурации** - Поддержка нескольких форматов конфигурации

### Phase 2: Core Features (Weeks 3-4)
1. **Система обработки ошибок** - Управление ошибками с помощью структурированных ошибок
2. **Основы наблюдаемости** - Структурированное логирование и метрики
3. **Улучшения безопасности** - Валидация входных данных и ограничение скорости
4. **Оптимизация производительности** - Пул соединений

### Phase 3: Advanced Features (Weeks 5-6)
1. **Распределенное трассирование** - Интеграция OpenTelemetry
2. **Расширенный кэш** - Интеграция Redis
3. **Асинхронная обработка** - Обработка задач в фоне
4. **Полное тестирование** - Интеграционные и конечные тесты

### Phase 4: Polish & Documentation (Weeks 7-8)
1. **Performance benchmarking** - Load testing and optimization
2. **Security audit** - Security testing and hardening
3. **Documentation updates** - API docs and guides
4. **Migration guides** - Upgrade documentation

## Success Metrics

### Code Quality
- **Test coverage**: Target 85%+ coverage
- **Code complexity**: Reduce cyclomatic complexity by 30%
- **Technical debt**: Eliminate high-priority technical debt items

### Performance
- **Memory usage**: Reduce memory footprint by 25%
- **CPU usage**: Reduce CPU overhead by 20%
- **Response time**: Improve command response time by 40%

### Maintainability
- **Build time**: Reduce build time by 30%
- **Test execution**: Reduce test suite time by 50%
- **Documentation**: 100% API documentation coverage

### Security
- **Vulnerabilities**: Zero high-severity vulnerabilities
- **Compliance**: Meet enterprise security standards
- **Audit readiness**: Complete security audit trail

## Conclusion

These enhancements will transform ServerEye into a truly enterprise-grade monitoring system with modern architectural patterns, comprehensive observability, and robust security. The phased approach ensures incremental improvements while maintaining system stability.

The focus on dependency injection, interface-driven design, and comprehensive testing will significantly improve code maintainability and developer experience. Performance optimizations and security enhancements will ensure the system can handle enterprise workloads safely and efficiently.

By implementing these improvements, ServerEye will be well-positioned for long-term growth and adoption in enterprise environments.
