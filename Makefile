# =========================
# EchoPBX Gateway — Makefile (Ubuntu ARM64 only, SAFE)
# =========================

APP            := echopbx-gateway
GOOS           := linux
GOARCH         := arm64
CGO_ENABLED    := 0

# FHS
PREFIX         ?= /usr
BINDIR         := $(PREFIX)/bin
ETCDIR         := /etc/echopbx
UNITDIR        := /lib/systemd/system

# Build
BUILD_DIR      := dist
OUT_ARM64      := $(BUILD_DIR)/linux-$(GOARCH)/$(APP)

# Versión embebida (si no hay git describe, fija 0.1.0)
DEB_VERSION    := $(shell git describe --tags --dirty --always 2>/dev/null || echo 0.1.0)
LDFLAGS        := -s -w -X main.version=$(DEB_VERSION)

.PHONY: all
all: build-linux-arm64

# ------------ Build ------------
$(OUT_ARM64): go.mod
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -trimpath -ldflags "$(LDFLAGS)" \
		-o $(OUT_ARM64) ./cmd/$(APP)

.PHONY: build-linux-arm64
build-linux-arm64: $(OUT_ARM64)

# ------------ Install ------------
.PHONY: install
install: $(OUT_ARM64)
	install -Dm755 $(OUT_ARM64) $(DESTDIR)$(BINDIR)/$(APP)
	install -Dm644 packaging/systemd/$(APP).service $(DESTDIR)$(UNITDIR)/$(APP).service
	install -Dm644 packaging/defaults/etc/echopbx/config.yaml $(DESTDIR)$(ETCDIR)/config.yaml
	install -Dm644 packaging/defaults/etc/echopbx/plugins.json $(DESTDIR)$(ETCDIR)/plugins.json
	@echo "Instalado en $(DESTDIR)$(BINDIR)/$(APP)"

.PHONY: uninstall
uninstall:
	- systemctl stop $(APP) || true
	- systemctl disable $(APP) || true
	- rm -f $(UNITDIR)/$(APP).service
	- rm -f $(BINDIR)/$(APP)
	@echo "Desinstalado (pueden quedar archivos en $(ETCDIR)/)"

# ------------ Service helpers ------------
.PHONY: service-enable
service-enable:
	systemctl daemon-reload
	systemctl enable --now $(APP)
	systemctl status $(APP) --no-pager

.PHONY: service-restart
service-restart:
	systemctl restart $(APP)
	systemctl status $(APP) --no-pager

.PHONY: service-stop
service-stop:
	systemctl stop $(APP)

.PHONY: logs
logs:
	journalctl -u $(APP) -f

.PHONY: sighup
sighup:
	kill -HUP `pidof $(APP)` || true

# ------------ Run (foreground) ------------
.PHONY: run
run: $(OUT_ARM64)
	@if [ ! -f "$(ETCDIR)/config.yaml" ]; then \
		echo ">> Creando $(ETCDIR)/config.yaml (fake_ari: true)"; \
		sudo mkdir -p $(ETCDIR); \
		sudo install -Dm644 packaging/defaults/etc/echopbx/config.yaml $(ETCDIR)/config.yaml; \
		sudo install -Dm644 packaging/defaults/etc/echopbx/plugins.json $(ETCDIR)/plugins.json; \
	fi
	$(OUT_ARM64)

# ------------ Utils ------------
.PHONY: sha256
sha256:
	@sha256sum $(OUT_ARM64) | tee $(OUT_ARM64).sha256

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
