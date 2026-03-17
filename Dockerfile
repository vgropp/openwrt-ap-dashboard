# Frontend
FROM node:24-bookworm@sha256:5a593d74b632d1c6f816457477b6819760e13624455d587eef0fa418c8d0777b AS frontend-build

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Backend
FROM golang:1.26-bookworm@sha256:4465644228bc2857a954b092167e12aa59c006a3492282a6c820bf4755fd64a4 AS backend-build

WORKDIR /app
COPY backend/ ./
RUN go mod download

# Set CGO disabled for static linking (distroless compatible)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o /go/bin/openwrt-dash .

# runtime
FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838

WORKDIR /app

COPY --from=backend-build /go/bin/openwrt-dash /app/openwrt-dash
COPY --from=frontend-build /app/frontend/dist /frontend/dist

USER nonroot:nonroot

ENTRYPOINT ["/app/openwrt-dash"]

