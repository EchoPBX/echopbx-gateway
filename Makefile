# =========================
# EchoPBX Gateway — Makefile (macOS -> Linux/ARM64)
# =========================

APP              := echopbx-gateway

GOOS             ?= linux
GOARCH           ?= arm64

CC_CROSS         := zig cc -target aarch64-linux-gnu
CXX_CROSS        := zig c++ -target aarch64-linux-gnu

BUILD_DIR        := dist
OUT_BIN          := $(BUILD_DIR)/$(GOOS)-$(GOARCH)/$(APP)

VERSION          := $(shell git describe --tags --dirty --always 2>/dev/null || echo 0.1.0)
LDFLAGS          := -s -w -X main.version=$(VERSION) -linkmode=internal

VM_HOST          ?=
VM_USER          ?= echopbx
VM_BIN_DST       ?= /usr/local/bin/$(APP)
VM_CFG_DIR       ?= /etc/echopbx

CFG_SRC          := packaging/defaults/etc/echopbx/config.yaml
PLUGINS_SRC      := packaging/defaults/etc/echopbx/plugins.json
UNIT_SRC         := packaging/systemd/$(APP).service
UNIT_DST         := /lib/systemd/system/$(APP).service

# =========================
# Main targets
# =========================

.PHONY: build
build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 \
	CC="$(CC_CROSS)" CXX="$(CXX_CROSS)" \
	  go build -trimpath -ldflags "$(LDFLAGS)" \
	  -o $(OUT_BIN) ./cmd/$(APP)
	@echo ">> Build OK: $(OUT_BIN)"

.PHONY: build-safe
build-safe:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 \
	  go build -trimpath -ldflags "$(LDFLAGS)" \
	  -o $(OUT_BIN) ./cmd/$(APP)
	@echo ">> Build SAFE (CGO=0): $(OUT_BIN)"

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# =========================
# Local helpers
# =========================

.PHONY: sha256
sha256:
	@sha256sum $(OUT_BIN) | tee $(OUT_BIN).sha256

# =========================
# Despliegue en VM (sin .deb)
# Requiere: VM_HOST=... (y acceso SSH)
# =========================

.PHONY: vm-copy-bin
vm-copy-bin: build
	@if [ -z "$(VM_HOST)" ]; then echo "ERROR: define VM_HOST=..." && exit 1; fi
	scp $(OUT_BIN) $(VM_USER)@$(VM_HOST):/home/$(VM_USER)/$(APP)
	@echo ">> Binario copiado: /home/$(VM_USER)/$(APP)"

.PHONY: vm-install-bin
vm-install-bin: vm-copy-bin
	ssh $(VM_USER)@$(VM_HOST) "sudo install -m755 /home/$(VM_USER)/$(APP) $(VM_BIN_DST)"
	@echo ">> Binario instalado: $(VM_BIN_DST)"

.PHONY: vm-install-config
vm-install-config:
	@if [ -z "$(VM_HOST)" ]; then echo "ERROR: define VM_HOST=..." && exit 1; fi
	ssh $(VM_USER)@$(VM_HOST) "sudo mkdir -p $(VM_CFG_DIR)"
	ssh $(VM_USER)@$(VM_HOST) "test -f $(VM_CFG_DIR)/config.yaml || sudo install -m644 $(CFG_SRC) $(VM_CFG_DIR)/config.yaml"
	ssh $(VM_USER)@$(VM_HOST) "test -f $(VM_CFG_DIR)/plugins.json || sudo install -m644 $(PLUGINS_SRC) $(VM_CFG_DIR)/plugins.json"
	@echo ">> Config por defecto en $(VM_CFG_DIR)"

.PHONY: vm-install-unit
vm-install-unit:
	@if [ -z "$(VM_HOST)" ]; then echo "ERROR: define VM_HOST=..." && exit 1; fi
	ssh $(VM_USER)@$(VM_HOST) "sudo install -m644 $(UNIT_SRC) $(UNIT_DST) && sudo systemctl daemon-reload"
	@echo ">> Systemd unit instalada: $(UNIT_DST)"

.PHONY: vm-restart
vm-restart:
	@if [ -z "$(VM_HOST)" ]; then echo "ERROR: define VM_HOST=..." && exit 1; fi
	# Si existe systemd unit, reinicia; si no, ejecuta foreground con config
	ssh $(VM_USER)@$(VM_HOST) '\
	  if systemctl list-unit-files | grep -q "$(APP).service"; then \
	    sudo systemctl restart $(APP) && sudo systemctl status $(APP) --no-pager; \
	  else \
	    echo ">> No hay unit; ejecutando en foreground (Ctrl+C para parar)"; \
	    ECHO_PBX_CONFIG=$(VM_CFG_DIR)/config.yaml $(VM_BIN_DST); \
	  fi'

.PHONY: vm-sighup
vm-sighup:
	@if [ -z "$(VM_HOST)" ]; then echo "ERROR: define VM_HOST=..." && exit 1; fi
	ssh $(VM_USER)@$(VM_HOST) "sudo systemctl kill -s HUP $(APP) || sudo kill -HUP \`pidof $(APP)\` || true"
	@echo ">> SIGHUP enviado"

.PHONY: vm-logs
vm-logs:
	@if [ -z "$(VM_HOST)" ]; then echo "ERROR: define VM_HOST=..." && exit 1; fi
	ssh -t $(VM_USER)@$(VM_HOST) "sudo journalctl -u $(APP) -f -n 200 --no-pager"

# =========================
# Help
# =========================
.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
