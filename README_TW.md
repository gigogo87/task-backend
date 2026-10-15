# Task List API

[English](README.md) | 繁體中文

以 Go 撰寫的 RESTful 任務清單 API，分為 middleware / controller / service / repository 四層，並透過建構函式注入依賴（dependency injection）。

## 環境需求

- **Go 1.27 以上**（`go.mod` 指定的版本），**或**只要有 Docker：Docker 建置時會自帶 Go 工具鏈，不需要另外安裝任何東西。

### 關於 Go 版本

本專案選擇使用最新的 Go 版本，而不是遷就能支援的最舊版本：

| 使用的功能 | 最低 Go 版本 |
|---|---|
| `log/slog` 結構化 log | 1.21 |
| `net/http` ServeMux 的「方法 + 路徑參數」路由（`"PUT /tasks/{id}"`） | 1.22 |
| `slog.DiscardHandler`、`go.mod` 的 `tool` 指令（mockgen） | 1.24 |
| `go-playground/validator` 與 `golang.org/x/time` 的目前版本 | 1.26 |

實際影響：

- **已安裝 Go 1.21 以上：** `go run .`、`go test ./...` 可以直接執行。在預設的 `GOTOOLCHAIN=auto` 下，Go 會在第一次使用時自動下載 1.27 工具鏈（只有第一次會花一點時間）。
- **已安裝 Go 1.18～1.20：** 這些版本無法自動下載較新的工具鏈，請改用 Docker（見下方）或升級 Go。

## 執行

```bash
go run .
```

| 環境變數 | 預設值 | 說明 |
|---|---|---|
| `PORT` | `8080` | 監聽的 port |
| `RATE_LIMIT_RPS` | `10` | 每個 client IP 每秒可發送的請求數 |
| `RATE_LIMIT_BURST` | `20` | 每個 client IP 瞬間最多可連續發送的請求數 |

設定值不合法時，server 會在啟動時直接結束。log 以 JSON 格式逐行輸出到 stdout，與請求相關的 log 會帶有 `trace_id`。

### 使用 Docker

```bash
docker build -t taskapi .
docker run --rm -p 8080:8080 taskapi
```

## 測試

```bash
go test ./...                 # 執行所有測試
go test -race -cover ./...    # 加上 race detector（需要 cgo）與覆蓋率
go generate ./...             # 修改 interface 後重新產生 mock
```

service 與 controller 的測試使用 [mockgen](https://github.com/uber-go/mock) 依 `TaskRepository`、`TaskService` 兩個 interface 產生的 mock（產生在 `internal/repository/mocks` 與 `internal/service/mocks`）。mockgen 以 `tool` 依賴的形式記錄在 `go.mod`，所以 `go generate` 會使用固定的版本，不需要另外安裝。產生的檔案有一併 commit，請勿手動修改。

慣例：斷言使用 [testify](https://github.com/stretchr/testify)（前提條件用 `require`，其餘用 `assert`），測試名稱採用 `Test<函式>_<情境>_<預期結果>` 格式，每個測試分為 Arrange / Act / Assert 三段。

## API

一筆任務的格式為 `{"id": 1, "name": "buy milk", "status": 0}`，其中 `status` 為 `0`（未完成）或 `1`（已完成）。

| 方法 | 路徑 | Body | 成功時回應 |
|---|---|---|---|
| GET | `/tasks` | – | `200` `{"result": [Task, ...]}` |
| POST | `/tasks` | `{"name": "...", "status": 0}` | `201` `{"result": Task}` |
| PUT | `/tasks/{id}` | `{"name": "...", "status": 1}` | `200` `{"result": Task}` |
| DELETE | `/tasks/{id}` | – | `204`（任務不存在時也是） |

Body 規則：`name` 為必填，最多 100 字元；`status` 可省略，只能是 `0` 或 `1`，預設為 `0`。`PUT` 會整筆覆蓋任務，因此省略 `status` 會將它設回 `0`。不接受未定義的欄位，body 大小上限為 1 MB。

`DELETE` 是冪等的：它的意思是「確保這筆任務不存在」，所以即使任務從沒建立過或已經被刪除，也會回 `204`。這樣 client 在逾時後重試一個其實已經成功的刪除，會得到 `204`，而不是誤導人的 `404`。代價是刪除打錯的 id 時不會被提醒。`PUT` 遇到不存在的任務仍回 `404`，因為它是更新既有的任務，不是 upsert。

### Trace ID

每個回應都會帶有 `X-Trace-Id` header。如果請求帶了 `X-Trace-Id` header，就會沿用該值，方便和呼叫端的 log 對照；否則會產生一個新的 32 字元 hex id。只有在請求的值為 1～128 個字元、且只包含英數字與 `-`、`_`、`.` 時才會沿用（UUID 與 hex id 都符合）；其他值會被忽略並產生新的 id，因為 trace id 會原樣寫進 log 與回應 header。

### 錯誤

所有錯誤都使用同一種格式：

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "request validation failed",
    "details": [
      { "field": "status", "reason": "oneof", "param": "0 1" }
    ],
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
  }
}
```

- `code` 是固定不變、給程式判斷用的值；`message` 是給人看的說明，內容可能會調整。
- `details` 只會出現在 `VALIDATION_FAILED`；`reason` 是 validator 的 tag（`required`、`max`、`oneof`），`param` 是 tag 的參數。
- `trace_id` 與 `X-Trace-Id` header 相同，可以用來在 log 中找到這個請求。

| HTTP | `code` | 發生情境 |
|---|---|---|
| 400 | `INVALID_JSON` | JSON 格式錯誤、有未定義的欄位、型別錯誤、body 為空、JSON 物件後面還有多餘內容 |
| 400 | `VALIDATION_FAILED` | body 驗證失敗（見 `details`） |
| 400 | `INVALID_ID` | `{id}` 不是正整數 |
| 404 | `TASK_NOT_FOUND` | `PUT` 的任務 id 不存在（`DELETE` 則是回 `204`） |
| 404 | `ROUTE_NOT_FOUND` | 路徑不存在 |
| 405 | `METHOD_NOT_ALLOWED` | 該路徑不支援此 HTTP 方法（`Allow` header 會列出可用的方法） |
| 413 | `PAYLOAD_TOO_LARGE` | body 超過 1 MB |
| 429 | `RATE_LIMITED` | 此 IP 的請求過多（見 `Retry-After` header） |
| 500 | `INTERNAL_ERROR` | 非預期的錯誤；詳細內容只寫進 log，不會回傳給 client |

## 設計

```
main.go                          讀取設定、建立 DependencyProvider、啟動 server 並支援優雅關閉
internal/
├── app/
│   ├── config.go                以環境變數設定
│   ├── dependency_provider.go   唯一建立並串接物件的地方（DI）
│   └── router.go                路由 + middleware 串接
├── middleware/                  trace id、以 IP 為單位的限流、未匹配路由的 JSON 404/405
├── controller/                  解析並驗證請求、呼叫 service、寫出回應
├── dto/                         API 請求格式、validate tag、validator 設定
├── response/                    JSON / 錯誤回應的輸出、HTTP 層專屬的錯誤、錯誤碼 → HTTP status
├── service/                     商業邏輯，依賴 repository.TaskRepository
├── repository/                  儲存層，實作 TaskRepository interface（目前存在記憶體）
├── model/                       Task、AppError 與商業錯誤（不知道 HTTP 的存在）
└── trace/                       trace id 工具，以及自動加上 trace_id 的 slog handler
```

請求的處理順序：`Trace → RateLimit → RouteErrors → router → controller → service → repository`。

- **依賴注入**：每個 `NewXxx` 都透過參數取得依賴，不會自己建立其他層的物件。`app.NewDependencyProvider` 由下往上建立所有元件。要把記憶體儲存換成資料庫，只需要寫一個新的 `TaskRepository` 實作，並修改 provider 裡的一行。測試則直接注入 mock（`service` 測試注入 mock repository，`controller` 測試注入 mock service）。
- **格式驗證與商業規則**：格式檢查是 `dto.TaskRequest` 上的 validator tag，由 controller 執行。商業規則則屬於 service。例如 repository 一律把找不到的任務回報為 `TASK_NOT_FOUND`，再由 service 決定：對 `Delete` 來說這算成功（冪等），對 `Update` 來說則是錯誤。
- **錯誤處理**：各層回傳 `*model.AppError`，它只帶錯誤碼、不帶 HTTP status。`response.WriteError` 是唯一把錯誤轉成 JSON 的地方，並依 `response/errors.go` 的錯誤碼對照表決定 HTTP status。只會在 HTTP 層發生的錯誤（`INVALID_JSON`、`RATE_LIMITED` 等）定義在 `response`，因此 `model`、`service`、`repository` 都不知道 HTTP 的存在。其他非預期的錯誤一律回 500，並由 controller 寫進 log。
- **限流**：以 `golang.org/x/time/rate` 對每個 client IP 做 token bucket 限流，閒置的項目每分鐘清理一次。使用的是連線的來源位址，而不是 `X-Forwarded-For`，因此 client 無法偽造；如果放在可信任的 proxy 後面，這部分需要調整。狀態存在記憶體中，所以限制是以單一 instance 計算。
- **並行安全**：`MemoryTaskRepository` 以 `sync.RWMutex` 保護內部的 map；`TestCreate_ConcurrentCalls_StoresAllTasks` 搭配 `-race` 驗證這點。
- **優雅關閉**：收到 SIGINT/SIGTERM 時，server 停止接受新連線，並最多等待 5 秒讓處理中的請求完成。
- **Docker**：multi-stage build 產生精簡的 Alpine image，並以非 root 使用者執行。

### 若要上線，我會再加入

持久化儲存（PostgreSQL/MySQL）、`GET /tasks` 的分頁、access log、身分驗證、OpenTelemetry 匯出、OpenAPI 文件，以及執行測試與 lint 的 CI。
