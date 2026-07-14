# syntax=docker/dockerfile:1.7

FROM golang:1.26.5-alpine AS build

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG BRANCH=unknown

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
      -ldflags "-s -w \
      -X github.com/russlank/lang-forge/internal/version.Version=${VERSION} \
      -X github.com/russlank/lang-forge/internal/version.Commit=${COMMIT} \
      -X github.com/russlank/lang-forge/internal/version.BuildDate=${BUILD_DATE} \
      -X github.com/russlank/lang-forge/internal/version.Branch=${BRANCH}" \
      -o /out/lang-forge ./cmd/lang-forge

FROM alpine:3.20

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG BRANCH=unknown
ARG GIT_SHA=unknown
ARG GIT_BRANCH=unknown
ARG REPO_URL=unknown
ARG REPO_TYPE=git
ARG CI=false

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /workspace

COPY --from=build /out/lang-forge /usr/local/bin/lang-forge

ENV CI=${CI}

LABEL org.opencontainers.image.title="lang-forge" \
      org.opencontainers.image.description="Modern Go implementation of Lex/Yacc-style compiler tooling" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.source="${REPO_URL}" \
      org.opencontainers.image.vendor="Russlan Kafri" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.ref.name="${BRANCH}" \
      io.langforge.git_sha="${GIT_SHA}" \
      io.langforge.git_branch="${GIT_BRANCH}" \
      io.langforge.repo_type="${REPO_TYPE}" \
      io.langforge.ci="${CI}"

ENTRYPOINT ["/usr/local/bin/lang-forge"]
CMD ["version"]
