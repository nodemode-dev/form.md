VERSION := v0.47.1
LDFLAGS := -s -w -X github.com/nodemode-ai/form.md/internal/app.Version=$(VERSION)
GO_BUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"
TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build web notices server release check clean

build: web notices server

web:
	test -d web/node_modules || ./web/oc_install
	cd web && npm run build
	go run ./tools/bundlecheck

notices:
	go run ./tools/notices

server:
	$(GO_BUILD) -o bin/form-md ./cmd/form-md

release: web notices
	rm -rf release && mkdir -p release
	@for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; ext=""; \
		if [ "$$os" = windows ]; then ext=".exe"; fi; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch $(GO_BUILD) -o release/form-md_$(VERSION)_$${os}_$${arch}$$ext ./cmd/form-md || exit 1; \
	done
	cd release && sha256sum form-md_* > SHA256SUMS

check:
	gofmt -l . | (! grep .)
	go vet ./...
	go run ./tools/wikicheck
	go run ./tools/skillcheck
	cd web && npm run typecheck

clean:
	rm -rf bin release web/dist/assets web/dist/index.html web/dist/favicon.svg
