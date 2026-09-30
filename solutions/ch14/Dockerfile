# syntax=docker/dockerfile:1

# Stage 1: compile. This stage is large (the Go toolchain, the module cache,
# the source) and none of it ships.
ARG GO_VERSION=1.27.1
FROM golang:${GO_VERSION} AS build
WORKDIR /src

# Copy the module files first and download dependencies in their own layer:
# as long as go.mod and go.sum are unchanged, later edits to the source reuse
# this layer instead of downloading everything again.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
# CGO_ENABLED=0 produces a statically linked binary with no dependency on the
# C library, so it runs in an image that has no C library. -trimpath removes
# build-machine paths from the binary; -s -w drop debug tables to save space.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ \
        ./cmd/aggregatord ./cmd/feedbackd ./cmd/policyd ./cmd/banditrpc

# Stage 2: what every service image starts from. "distroless/static" holds
# CA certificates and little else: no shell, no package manager. The ":nonroot"
# tag runs as an unprivileged user by default.
FROM gcr.io/distroless/static-debian12:nonroot AS base
# banditrpc doubles as the health checker: `banditrpc health` exits non-zero
# unless the service answers SERVING.
COPY --from=build /out/banditrpc /banditrpc

# One final stage per service; build one with `--target policyd`. The
# ENTRYPOINT is in exec form (a JSON array) so the binary is process 1 and
# receives SIGTERM directly when the container is stopped. With the shell form
# a shell would be process 1 and the signal would never reach the service.
FROM base AS aggregatord
COPY --from=build /out/aggregatord /aggregatord
EXPOSE 9092
ENTRYPOINT ["/aggregatord"]

FROM base AS feedbackd
COPY --from=build /out/feedbackd /feedbackd
EXPOSE 9091
ENTRYPOINT ["/feedbackd"]

FROM base AS policyd
COPY --from=build /out/policyd /policyd
EXPOSE 9090
ENTRYPOINT ["/policyd"]
