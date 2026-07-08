FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/backend/cmd/kkcert/dist ./cmd/kkcert/dist
RUN CGO_ENABLED=0 go build -buildvcs=false -o /kkcert ./cmd/kkcert

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git openssh-client tzdata
ENV TZ=Asia/Shanghai
WORKDIR /data
COPY --from=backend /kkcert /usr/local/bin/kkcert
ENV KKCERT_DATA_DIR=/data KKCERT_LISTEN=:8080
EXPOSE 8080
ENTRYPOINT ["kkcert"]
