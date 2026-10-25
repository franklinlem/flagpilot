FROM golang:1.27.3-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS gosu-build
WORKDIR /src
COPY deploy/postgres/gosu/go.mod deploy/postgres/gosu/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /gosu github.com/tianon/gosu
FROM postgres:16.15-alpine@sha256:f4610795587492cd4fb887c2dc3de301b813278bb6f616992061141b5f15ac36 AS runtime
# Mantém entrypoint/identidade do upstream; recompila apenas seu helper com Go corrigido.
COPY --from=gosu-build /gosu /usr/local/bin/gosu
