# syntax=docker/dockerfile:1

# ── Build ────────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS build

WORKDIR /src

# Dependencies first, so a source-only change does not re-download the module
# cache on every build.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=docker
ARG COMMIT=unknown
ARG DATE=unknown

# CGO off keeps the binary static, so it runs on a base image with no libc of
# the builder's vintage. -trimpath keeps build paths out of the binary.
RUN CGO_ENABLED=0 go build \
        -trimpath \
        -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
        -o /out/netdiag .

# ── Capabilities ─────────────────────────────────────────────────────────────
# setcap has to run somewhere with libcap, and the final image should not carry
# a package manager just to install it. Do it in its own stage and copy the
# binary with its file capabilities intact.
FROM alpine:3.21 AS setcap

RUN apk add --no-cache libcap
COPY --from=build /out/netdiag /out/netdiag

# cap_net_raw is what ping, trace, discover and `scan --fast` need. Granting it
# to the binary means the container does not have to run as root, and the
# capability cannot leak to anything else in the image.
RUN setcap cap_net_raw+ep /out/netdiag

# ── Runtime ──────────────────────────────────────────────────────────────────
# Alpine rather than distroless: file capabilities need a filesystem that
# preserves extended attributes through COPY, and having a shell in the image
# is worth more than the few MB for a tool people will want to exec into.
FROM alpine:3.21

# ca-certificates for the HTTPS commands (http, speedtest); the rest of netdiag
# speaks raw TCP, UDP and ICMP and needs nothing else.
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 netdiag

COPY --from=setcap /out/netdiag /usr/local/bin/netdiag

# Unprivileged. The binary carries exactly the one capability it needs, so
# there is no reason for the process to be root.
#
# Note for hardened deployments: `--cap-drop=ALL` alone will not start this
# image. Linux refuses to exec a file with permitted capabilities the process
# could never be granted, so dropping cap_net_raw produces an exec error rather
# than a netdiag that falls back to unprivileged scanning. Drop everything and
# add back the one capability instead:
#
#     docker run --rm --cap-drop=ALL --cap-add=NET_RAW netdiag scan host --fast
USER netdiag

ENTRYPOINT ["/usr/local/bin/netdiag"]
CMD ["--help"]
