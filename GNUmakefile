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
	TF_ACC=1 go test ./internal/acceptance/... -v -run "^TestAcc" -timeout 300s

# testacc-create creates resources via the API and writes their IDs to
# acc-manual-state.json so they can be inspected in the Serverguard24 UI.
# testacc-destroy reads that file and deletes the resources.
testacc-create:
	TF_ACC=1 SG24_TEST_MODE=create SG24_TEST_STATEFILE=$(shell pwd)/acc-manual-state.json \
	  go test ./internal/acceptance/... -v -run "^TestAccManual_Create$$" -timeout 60s

testacc-destroy:
	TF_ACC=1 SG24_TEST_MODE=destroy SG24_TEST_STATEFILE=$(shell pwd)/acc-manual-state.json \
	  go test ./internal/acceptance/... -v -run "^TestAccManual_Destroy$$" -timeout 60s

lint:
	golangci-lint run ./...

generate:
	go generate ./...

tidy:
	go mod tidy

.PHONY: build install test testacc testacc-create testacc-destroy lint generate tidy
