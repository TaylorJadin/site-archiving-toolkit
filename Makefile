.PHONY: build run test vet fmt releases clean

BINARY := archive
CMD := ./cmd/archive
RELEASE_ROOT := releases

# Build the local binary.
build:
	go build -o $(BINARY) $(CMD)

# Run without building a binary first. Pass args via ARGS, e.g.:
#   make run
#   make run ARGS='https://example.com'
#   make run ARGS='resume'
run:
	go run $(CMD) $(ARGS)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

# Cross-compile release zips for each supported platform.
# Runtime assets are embedded, so each release is just the binary plus the README.
releases:
	rm -rf $(RELEASE_ROOT)
	mkdir -p $(RELEASE_ROOT)
	@set -e; \
	for target in \
		"linux amd64 Linux" \
		"darwin amd64 macOS-Intel" \
		"darwin arm64 macOS-AppleSilicon" \
		"windows amd64 Windows .exe"; \
	do \
		set -- $$target; \
		goos=$$1; goarch=$$2; label=$$3; ext=$${4:-}; \
		dir=$(RELEASE_ROOT)/site-archiving-toolkit; \
		echo "Building $${label} ($${goos}/$${goarch})..."; \
		mkdir -p "$$dir"; \
		GOOS="$$goos" GOARCH="$$goarch" CGO_ENABLED=0 go build -o "$$dir/$(BINARY)$$ext" $(CMD); \
		cp README.md "$$dir/"; \
		(cd $(RELEASE_ROOT) && zip -qr "$${label}-site-archiving-toolkit.zip" site-archiving-toolkit); \
		rm -rf "$$dir"; \
		echo "Wrote $(RELEASE_ROOT)/$${label}-site-archiving-toolkit.zip"; \
	done
	@echo "Done."

clean:
	rm -f $(BINARY)
	rm -rf $(RELEASE_ROOT)
