SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

.PHONY: help test verify mod-tidy-check govulncheck smoke firmware-smoke quickstart run-ui run-daemon ci-local release-readiness

help:
	@echo "MAMA developer commands"
	@echo "  make test            - Run Go tests"
	@echo "  make verify          - Verify Go module cache"
	@echo "  make mod-tidy-check  - Ensure go.mod/go.sum are tidy"
	@echo "  make govulncheck     - Run vulnerability scan"
	@echo "  make smoke           - Run quickstart smoke test"
	@echo "  make firmware-smoke  - Run firmware protocol stress tests"
	@echo "  make quickstart      - Build dist/mama-quickstart bundle"
	@echo "  make run-ui          - Start setup UI"
	@echo "  make run-daemon      - Start runtime daemon"
	@echo "  make ci-local        - Run local CI-equivalent checks"
	@echo "  make release-readiness - Run production readiness gate checks"

test:
	cd mama && go test ./...

verify:
	cd mama && go mod verify

mod-tidy-check:
	cd mama && go mod tidy && git diff --exit-code -- go.mod go.sum

govulncheck:
	cd mama && go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...

smoke:
	scripts/quickstart-smoke-test.sh

firmware-smoke:
	scripts/firmware/run_encoder_stress_test.sh
	scripts/firmware/run_i2c_robustness_test.sh

quickstart:
	scripts/quickstart.sh

run-ui:
	cd mama && go run ./cmd/mama-ui

run-daemon:
	cd mama && go run ./cmd/mama

ci-local: test verify mod-tidy-check govulncheck

release-readiness:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/release/production-readiness-check.ps1
else
	scripts/release/production-readiness-check.sh
endif
