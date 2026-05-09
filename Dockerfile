# Frontend
FROM node:24-bookworm@sha256:34f0eb9d36f5163c16e8157aaa50c3bbb7a03aa744ce8668549d6f71d98b69cf AS frontend-build

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Backend
FROM golang:1.26-bookworm@sha256:b09e568dcf2a1ff3ce09a230ea234193fc014dc195472fe63316e50238453d96 AS backend-build

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

