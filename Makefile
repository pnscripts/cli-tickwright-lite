VERSION := 1.0.0
NAME    := tickwright-lite
LDFLAGS := -s -w -X main.version=$(VERSION)
TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: all test build dist images clean

all: test dist

test:
	go vet ./...
	go test ./...

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o build/$(NAME) ./cmd/$(NAME)

# GitHub release assets: one archive per platform plus SHA256SUMS.
dist:
	rm -rf dist build/pkg && mkdir -p dist
	@for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		dir=build/pkg/$(NAME)-$(VERSION)-$$os-$$arch; mkdir -p $$dir; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$dir/$(NAME)$$ext ./cmd/$(NAME) || exit 1; \
		cp LICENSE README.md CHANGELOG.md $$dir/; \
		if [ $$os = windows ]; then (cd build/pkg && zip -qr -X ../../dist/$(NAME)-$(VERSION)-$$os-$$arch.zip $(NAME)-$(VERSION)-$$os-$$arch); \
		else tar -C build/pkg -czf dist/$(NAME)-$(VERSION)-$$os-$$arch.tar.gz $(NAME)-$(VERSION)-$$os-$$arch; fi; \
	done
	cd dist && sha256sum $(NAME)-* > SHA256SUMS

images:
	node tools/render.mjs tools/images.mjs

clean:
	rm -rf build dist
