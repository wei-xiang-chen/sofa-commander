# 多階段構建 Dockerfile - 前後端整合
FROM node:18-alpine AS frontend-builder

# 設置工作目錄
WORKDIR /app/frontend

# 複製前端 package 檔案
COPY frontend/package*.json ./

# 安裝前端依賴
RUN npm ci --only=production

# 複製前端源碼
COPY frontend/ ./

# 構建前端
RUN npm run build

# Go 後端構建階段
FROM golang:1.23.1-alpine AS backend-builder

# 設置工作目錄
WORKDIR /app

# 安裝必要的系統依賴
RUN apk add --no-cache git

# 複製 go mod 檔案
COPY backend/go.mod backend/go.sum ./

# 下載依賴
RUN go mod download

# 複製源碼
COPY backend/ ./

# 構建應用程式
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 最終運行階段
FROM alpine:latest

# 安裝 ca-certificates 和 nginx
RUN apk --no-cache add ca-certificates nginx

# 創建非 root 用戶
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 設置工作目錄
WORKDIR /app

# 從 backend-builder 階段複製編譯好的應用程式
COPY --from=backend-builder /app/main .

# 從 frontend-builder 階段複製構建好的前端檔案
COPY --from=frontend-builder /app/frontend/build ./static

# 複製配置檔案
COPY backend/config/ ./config/

# 創建 nginx 配置
RUN mkdir -p /etc/nginx/http.d
COPY <<EOF /etc/nginx/http.d/default.conf
server {
    listen 80;
    server_name localhost;
    root /app/static;
    index index.html;

    # 處理 React Router
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API 路由代理到後端
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # 健康檢查
    location /ping {
        proxy_pass http://localhost:8080/ping;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # 靜態資源快取
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# 創建 nginx 用戶配置
COPY <<EOF /etc/nginx/nginx.conf
user appuser appgroup;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /tmp/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format main '\$remote_addr - \$remote_user [\$time_local] "\$request" '
                    '\$status \$body_bytes_sent "\$http_referer" '
                    '"\$http_user_agent" "\$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    
    include /etc/nginx/http.d/*.conf;
}
EOF

# 創建啟動腳本
COPY <<EOF /app/start.sh
#!/bin/sh
# 啟動後端服務
./main &
BACKEND_PID=\$!

# 啟動 nginx
nginx -g "daemon off;" &
NGINX_PID=\$!

# 等待任一進程結束
wait \$BACKEND_PID \$NGINX_PID
EOF

# 設置權限
RUN chmod +x /app/start.sh && \
    chown -R appuser:appgroup /app && \
    chown -R appuser:appgroup /etc/nginx && \
    chown -R appuser:appgroup /var/log/nginx && \
    chown -R appuser:appgroup /var/lib/nginx && \
    chown -R appuser:appgroup /tmp

# 切換到非 root 用戶
USER appuser

# 暴露端口
EXPOSE 80

# 健康檢查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ping || exit 1

# 啟動應用程式
CMD ["/app/start.sh"] 