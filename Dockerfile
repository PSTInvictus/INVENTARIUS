# One image: the Go server with the built frontend embedded.

FROM node:22-alpine AS frontend
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.27-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/inventarius ./cmd/server
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /out/inventarius /inventarius
COPY --from=backend --chown=nonroot:nonroot /out/data /data
ENV INVENTARIUS_DATA_DIR=/data INVENTARIUS_ADDR=:8080
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/inventarius"]
