# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM node:22.22.0-bookworm-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
COPY demo/index.html /src/demo/index.html
COPY demo/assets /src/demo/assets
COPY demos/views/index.html /src/demos/views/index.html
COPY demos/views/assets /src/demos/views/assets
COPY demos/publish/index.html /src/demos/publish/index.html
COPY demos/publish/assets /src/demos/publish/assets
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26.9-bookworm AS server
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w -X github.com/nodemode-ai/form.md/internal/app.Version=$VERSION" \
      -o /out/form-md ./cmd/form-md \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/form-md /usr/local/bin/form-md
COPY --from=server --chown=65532:65532 /out/data /data
ENV FORMMD_DATA=/data \
    FORMMD_ADDR=0.0.0.0:8750
VOLUME ["/data"]
EXPOSE 8750
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/form-md"]
CMD ["serve"]
