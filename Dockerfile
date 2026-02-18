FROM node:22-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /build/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build -o /pxbin ./cmd/pxbin/

FROM alpine:3.21
RUN addgroup -g 1001 pxbin && adduser -u 1001 -G pxbin -D pxbin
COPY --from=backend /pxbin /pxbin
USER pxbin
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/pxbin"]
