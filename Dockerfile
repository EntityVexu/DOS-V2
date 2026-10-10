# ---------- Stage 1: Build Go worker ----------
FROM golang:1.22-alpine AS go-builder
WORKDIR /build
COPY vexu1.go .
RUN go mod init vexu-worker && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o vexu-worker vexu1.go

# ---------- Stage 2: Node + Go runtime ----------
FROM node:20-alpine

RUN apk add --no-cache libc6-compat

WORKDIR /app

COPY package*.json ./
RUN npm install --omit=dev

COPY server.js worker.js config.json ./
COPY public ./public

COPY --from=go-builder /build/vexu-worker ./vexu-worker
RUN chmod +x ./vexu-worker

ENV NODE_ENV=production
ENV PORT=3000

EXPOSE 3000

CMD ["node", "server.js"]
