# Frontend
FROM node:24-bookworm@sha256:934240a162082fd8b8a2f90cd5114446443f1eba1c5378f6687167ca405e6584 AS frontend-build

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Backend
FROM golang:1.26-bookworm@sha256:53eeac89074db483fdf0ab3be1df32bf6e47562263d2d0d6baa7f26acb4957dd AS backend-build

WORKDIR /app
COPY backend/ ./
RUN go mod download

# Set CGO disabled for static linking (distroless compatible)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o /go/bin/openwrt-dash .

# runtime
FROM gcr.io/distroless/static-debian12@sha256:6447365a6337c3732f412d1b74357b30a633831955b2bc45552b0086be907687

WORKDIR /app

COPY --from=backend-build /go/bin/openwrt-dash /app/openwrt-dash
COPY --from=frontend-build /app/frontend/dist /frontend/dist

USER nonroot:nonroot

ENTRYPOINT ["/app/openwrt-dash"]

