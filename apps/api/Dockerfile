FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/ ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server &&     CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck
FROM scratch AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /server /server
COPY --from=build /healthcheck /healthcheck
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
