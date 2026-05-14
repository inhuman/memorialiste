# Stage 1: build Go binary
FROM golang:1.26-alpine AS builder
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-X 'github.com/inhuman/memorialiste/cliconfig.Version=${VERSION}'" \
    -o /memorialiste ./cmd/memorialiste

# Stage 2: install Python deps
# distroless/python3-debian12 ships Python 3.11, so we match that here.
# The pinned trio below is the known-working combo for grep-ast's TreeContext
# rich rendering API. Newer versions break the TreeContext code path.
FROM python:3.11-slim AS python-deps
RUN pip install --no-cache-dir \
    "tree-sitter==0.20.4" \
    "tree-sitter-languages==1.10.2" \
    "grep-ast==0.5.0"

# Stage 3: distroless runtime — no shell, no package manager, minimal attack surface
FROM gcr.io/distroless/python3-debian12
# Copy pip-installed packages into the path Python will find them
COPY --from=python-deps /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /memorialiste /usr/local/bin/memorialiste
ENV PYTHONPATH=/usr/local/lib/python3.11/site-packages
ENTRYPOINT ["/usr/local/bin/memorialiste"]
