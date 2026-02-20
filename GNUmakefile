PLUGIN_DIR := $(HOME)/.terraform.d/plugins/registry.terraform.io/dg-i/serverguard24/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
DEV_OVERRIDES_FILE := $(HOME)/.terraformrc

default: build

build:
	go build -o terraform-provider-serverguard24 .

install: build
	mkdir -p $(PLUGIN_DIR)
	cp terraform-provider-serverguard24 $(PLUGIN_DIR)/

# dev installs the binary and writes a ~/.terraformrc with dev_overrides so that
# tofu/terraform uses the local build without checksum validation (no tofu init needed).
dev: build
	@mkdir -p $(HOME)/go/bin
	@cp terraform-provider-serverguard24 $(HOME)/go/bin/terraform-provider-serverguard24
	@printf 'provider_installation {\n  dev_overrides {\n    "registry.terraform.io/dg-i/serverguard24" = "%s/go/bin"\n  }\n  direct {}\n}\n' "$(HOME)" > $(DEV_OVERRIDES_FILE)
	@echo "Dev override written to $(DEV_OVERRIDES_FILE). Use 'tofu plan' directly (no init needed)."

test:
	go test ./... -v -timeout 120s

testacc:
	TF_ACC=1 go test ./... -v -run "^TestAcc" -timeout 120s

lint:
	golangci-lint run ./...

generate:
	go generate ./...

tidy:
	go mod tidy

.PHONY: build install test testacc lint generate tidy
