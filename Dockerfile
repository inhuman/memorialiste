# Stage 1: build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN CGO_ENABLED=0 go build -o /memorialiste ./cmd/memorialiste

# Stage 2: install Python deps
# distroless/python3-debian12 ships Python 3.11, so we match that here
FROM python:3.11-slim AS python-deps
RUN pip install --no-cache-dir grep-ast==0.9.0

# Stage 3: distroless runtime — no shell, no package manager, minimal attack surface
FROM gcr.io/distroless/python3-debian12
# Copy pip-installed packages into the path Python will find them
COPY --from=python-deps /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /memorialiste /usr/local/bin/memorialiste
ENV PYTHONPATH=/usr/local/lib/python3.11/site-packages
ENTRYPOINT ["/usr/local/bin/memorialiste"]
