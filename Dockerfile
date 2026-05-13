# Stage 1: build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN CGO_ENABLED=0 go build -o /memorialiste ./cmd/memorialiste

# Stage 2: runtime with Python + grep-ast
FROM python:3.12-alpine
RUN pip install --no-cache-dir \
    grep-ast==0.8.4 \
    tree-sitter-language-pack==0.3.4
COPY --from=builder /memorialiste /usr/local/bin/memorialiste
ENTRYPOINT ["memorialiste"]
