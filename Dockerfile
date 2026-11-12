# Frontend
FROM node:24-bookworm@sha256:9d266751ca3d7fdbe651c878ffd5ad0a42fed49a18c283509267c97f025b7496 AS frontend-build

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Backend
FROM golang:1.27-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS backend-build

WORKDIR /app
COPY backend/ ./
RUN go mod download

# Set CGO disabled for static linking (distroless compatible)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o /go/bin/openwrt-dash .

# runtime
FROM gcr.io/distroless/static-debian12@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

WORKDIR /app

COPY --from=backend-build /go/bin/openwrt-dash /app/openwrt-dash
COPY --from=frontend-build /app/frontend/dist /frontend/dist

USER nonroot:nonroot

ENTRYPOINT ["/app/openwrt-dash"]

