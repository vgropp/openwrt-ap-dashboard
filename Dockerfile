# Frontend
FROM node:26-bookworm@sha256:b7087bc4d973034b30854b378ba107f9fab622b47514cb7e2592e41b8bab8d49 AS frontend-build

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Backend
FROM golang:1.27-bookworm@sha256:ce59b41415e2524ae99926fe1f37ce48ba8c194dcbd2a56a65436cdcefa2bd34 AS backend-build

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

