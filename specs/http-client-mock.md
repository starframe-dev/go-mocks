# HTTP Client Mock

## Контекст

Проект `go-mocks` — библиотека моков для Go. Уже реализованы моки для файловой системы (`fs/`) и терминала (`tui/`). Нет мока для HTTP-клиента, что мешает тестировать код, делающий сетевые запросы, без реальных вызовов.

## Цель

Добавить пакет `net/` с моком HTTP-клиента: интерфейс в `contracts/net.go`, in-memory реализацию и callback-based мок в `net/`, полный набор unit-тестов по TDD.

## Что изменится

1. `contracts/net.go` — интерфейс `NetContract` для HTTP-операций + compile-time assertion
2. `net/mock.go` — `MockHTTPClient` с callback-функциями (`DoFunc`, `GetFunc`, `PostFunc`)
3. `net/client.go` — интерфейс `HTTPClient` + типы `Request`, `Response`
4. `net/inmemory.go` — `InMemoryHTTPClient` для интеграционных тестов с историей и матчингом по URL
5. `net/mock_test.go` — unit-тесты всех реализаций (TDD: сначала тесты, потом код)

## Детали реализации

### Интерфейс `HTTPClient`

```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
    Get(url string) (*http.Response, error)
    Post(url string, contentType string, body io.Reader) (*http.Response, error)
}
```

### `MockHTTPClient`

- Поля `DoFunc`, `GetFunc`, `PostFunc` — callback-функции
- Поле `history []RecordedRequest` — история всех вызовов
- Thread-safe через `sync.RWMutex`
- `History() []RecordedRequest` — копия истории
- `Reset()` — очистка истории

### `InMemoryHTTPClient`

- `Register(method, url, statusCode, body, headers)` — регистрация ответа для матчинга
- `Do(req)` — матчит по методу + URL, возвращает зарегистрированный ответ
- `Get(url)` — shorthand для GET
- `Post(url, ...)` — shorthand для POST
- История всех запросов
- Thread-safe

### `RecordedRequest`

```go
type RecordedRequest struct {
    Method  string
    URL     string
    Headers http.Header
    Body    []byte
}
```

### Тесты (TDD)

**Phase 1 — MockHTTPClient (unit):**
- `TestMockHTTPClient_Do` — вызов `Do` с callback
- `TestMockHTTPClient_DoFuncError` — callback возвращает ошибку
- `TestMockHTTPClient_Get` — вызов `Get`
- `TestMockHTTPClient_Post` — вызов `Post` с body
- `TestMockHTTPClient_History` — записывает историю запросов
- `TestMockHTTPClient_HistoryEmpty` — пустая история
- `TestMockHTTPClient_ConcurrentAccess` — race-free

**Phase 2 — InMemoryHTTPClient (unit):**
- `TestInMemoryHTTPClient_RegisterAndDo` — регистрация + вызов
- `TestInMemoryHTTPClient_DoNotFound` — запрос без регистрации → ошибка
- `TestInMemoryHTTPClient_Get` — shorthand GET
- `TestInMemoryHTTPClient_Post` — shorthand POST
- `TestInMemoryHTTPClient_History` — история
- `TestInMemoryHTTPClient_MultipleEndpoints` — разные URL
- `TestInMemoryHTTPClient_ConcurrentAccess` — race-free

**Phase 3 — Example Service (integration):**
- Сервис `UserService` с методами `FetchUser(id)` и `CreateUser(name)`
- `TestUserService_FetchUser_Success` — MockHTTPClient возвращает JSON, сервис парсит
- `TestUserService_FetchUser_Error` — MockHTTPClient возвращает ошибку, сервис обрабатывает
- `TestUserService_CreateUser_Success` — InMemoryHTTPClient с зарегистрированным ответом
- `TestUserService_CreateUser_VerifyRequest` — проверяем, что сервис отправил правильный POST

**Phase 4 — Contracts:**
- `TestNetContract_CompileTime` — compile-time assertion (var _ NetContract = ...)

## Критерии приёмки

- [x] `go test ./net/...` — все тесты проходят (21/21 PASS)
- [x] `go build ./...` — проект собирается без ошибок
- [x] `go vet ./...` — без предупреждений
- [x] Race detector: `go test -race ./net/...` — чисто
- [x] Покрытие `net/` ≥ 90% (фактически: **95.5%**)
- [x] Интерфейс `HTTPClient` совместим со `http.Client` из stdlib (можно подменять)
