# Sofa Commander - Docker 部署指南

## 🐳 Docker 快速開始

### 使用 Docker Compose（推薦）

1. **克隆專案**
   ```bash
   git clone <your-repo-url>
   cd sofa-commander
   ```

2. **設置環境變數**
   ```bash
   cp .env.example .env
   # 編輯 .env 檔案，設置 OPENAI_API_KEY
   ```

3. **啟動服務**
   ```bash
   docker-compose up -d
   ```

4. **訪問應用**
   - 前端應用: http://localhost
   - 後端 API: http://localhost/api
   - 健康檢查: http://localhost/ping

### 使用 Makefile

```bash
# 顯示所有可用命令
make help

# 構建 image
make build

# 開發模式（構建並運行）
make dev

# 查看日誌
make logs

# 停止服務
make stop

# 清理資源
make clean
```

### 手動 Docker 命令

```bash
# 構建 image
docker build -t sofa-commander .

# 運行 container
docker run -p 80:80 --env-file .env sofa-commander

# 在背景運行
docker run -d -p 80:80 --env-file .env --name sofa-commander sofa-commander
```

## 🔧 環境變數

創建 `.env` 檔案並設置以下變數：

```env
# OpenAI API 金鑰
OPENAI_API_KEY=your-openai-api-key

# Gin 模式（可選）
GIN_MODE=release
```

## 🚀 GitHub Actions CI/CD

專案已配置 GitHub Actions 自動化流程：

### 觸發條件
- **測試流程**：推送到 `main` 或 `develop` 分支，或 Pull Request 到 `main` 分支
- **構建和推送**：僅在創建版本標籤時（如 `v1.0.0`）

### 設置 GitHub Secrets

在 GitHub 倉庫設置以下 secrets：

1. `DOCKER_USERNAME`: 你的 Docker Hub 用戶名
2. `DOCKER_PASSWORD`: 你的 Docker Hub 密碼或 Access Token

### 自動化流程

#### 測試流程（分支推送/PR）
1. **代碼檢查**: Go vet 和測試
2. **前端測試**: React 測試
3. **Docker 構建測試**: 驗證 Docker image 可以正常構建和運行

#### 發布流程（版本標籤）
1. **構建**: 自動構建 Docker image
2. **測試**: 運行健康檢查
3. **推送**: 推送到 Docker Hub
4. **多架構**: 支援 linux/amd64 和 linux/arm64

## 📦 Docker Image 標籤

- `latest`: 最新版本（僅在 main 分支的版本標籤）
- `v1.0.0`: 完整版本號
- `v1.0`: 主次版本號

## 🔍 健康檢查

Docker image 包含健康檢查：

```bash
# 手動檢查
curl http://localhost/ping

# 查看健康狀態
docker inspect sofa-commander | grep Health -A 10
```

## 🛠️ 開發模式

### 本地開發

```bash
# 使用 docker-compose 進行開發
docker-compose up -d

# 修改配置檔案（會自動重新載入）
# 編輯 backend/config/app_config.json

# 查看日誌
docker-compose logs -f

# 訪問應用
# 前端: http://localhost
# API: http://localhost/api
```

### 調試

```bash
# 進入 container
docker exec -it sofa-commander /bin/sh

# 查看日誌
docker logs -f sofa-commander

# 檢查配置
docker exec sofa-commander cat /app/config/app_config.json
```

## 📋 部署檢查清單

- [ ] 設置 `.env` 檔案
- [ ] 配置 GitHub Secrets
- [ ] 測試本地構建
- [ ] 推送代碼觸發 CI/CD
- [ ] 驗證 Docker Hub 上的 image
- [ ] 測試部署

## 🔒 安全最佳實踐

1. **非 root 用戶**: Container 以非 root 用戶運行
2. **最小化基礎鏡像**: 使用 Alpine Linux
3. **多階段構建**: 減少最終 image 大小
4. **健康檢查**: 自動監控服務狀態
5. **環境變數**: 敏感資訊通過環境變數傳遞

## 📊 監控和日誌

```bash
# 查看資源使用
docker stats sofa-commander

# 查看詳細資訊
docker inspect sofa-commander

# 查看日誌
docker logs -f sofa-commander
```

## 🆘 故障排除

### 常見問題

1. **端口被佔用**
   ```bash
   # 檢查端口使用
   lsof -i :80
   
   # 停止佔用端口的服務
   docker stop $(docker ps -q)
   ```

2. **環境變數問題**
   ```bash
   # 檢查環境變數
   docker exec sofa-commander env
   ```

3. **配置檔案問題**
   ```bash
   # 檢查配置檔案
   docker exec sofa-commander cat /app/config/app_config.json
   ```

4. **網路問題**
   ```bash
   # 檢查網路連接
   docker exec sofa-commander wget -qO- http://localhost/ping
   ``` 