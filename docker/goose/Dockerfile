FROM docker.1ms.run/library/alpine:3.21

ARG TARGETARCH
ARG GOOSE_VERSION=1.44.0

RUN apk add --no-cache ca-certificates curl

# Download and install goose CLI
RUN case "${TARGETARCH}" in \
      amd64)  GOOSE_ARCH="x86_64-unknown-linux-musl" ;; \
      arm64)  GOOSE_ARCH="aarch64-unknown-linux-musl" ;; \
      *)      echo "Unsupported arch: ${TARGETARCH}"; exit 1 ;; \
    esac && \
    curl -sL "https://github.com/aaif-goose/goose/releases/download/v${GOOSE_VERSION}/goose-${GOOSE_ARCH}.tar.gz" \
      | tar xz -C /usr/local/bin && \
    chmod +x /usr/local/bin/goose

# Pre-create prompts directory for template mounting
RUN mkdir -p /root/.config/goose/prompts

WORKDIR /workspace
ENTRYPOINT ["goose"]
