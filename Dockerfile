# ---------- 第一階段：編譯 ----------
# 用有完整 Go 工具鏈的 image 來編譯；版本要 >= go.mod 裡的 go 版本
FROM golang:1.27-alpine AS builder

WORKDIR /app

# 先只複製 go.mod 和 go.sum，讓依賴下載這一層可以被 Docker 快取
COPY go.mod go.sum ./
RUN go mod download

# 再複製其餘原始碼並編譯
COPY . .
# CGO_ENABLED=0 產生不依賴 C 函式庫的靜態執行檔，才能放進極小的 image
RUN CGO_ENABLED=0 GOOS=linux go build -o /taskapi .

# ---------- 第二階段：執行 ----------
# 只放編譯好的執行檔，image 從約 300MB 降到約 10MB，也比較安全
FROM alpine:3.20

COPY --from=builder /taskapi /taskapi

# 不用 root 身分執行
USER nobody

EXPOSE 8080
ENTRYPOINT ["/taskapi"]
