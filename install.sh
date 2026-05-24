#!/usr/bin/env sh
set -e

BINARY_NAME="mycli"
INSTALL_DIR="/usr/local/bin"
GO_MIN_VERSION="1.24.1"
GO_CMD=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()    { printf "%b[INFO]%b %s\n" "$GREEN" "$NC" "$1"; }
warn()    { printf "%b[WARN]%b %s\n" "$YELLOW" "$NC" "$1"; }
error()   { printf "%b[ERROR]%b %s\n" "$RED" "$NC" "$1" >&2; exit 1; }

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        error "Este script precisa ser executado como root. Use: sudo ./install.sh"
    fi
}

version_ge() {
    # Retorna 0 se $1 >= $2
    printf '%s\n%s' "$2" "$1" | sort -C -V
}

check_go() {
    GO_CMD="$(command -v go || true)"

    if [ -z "$GO_CMD" ] && [ -x "/usr/local/go/bin/go" ]; then
        GO_CMD="/usr/local/go/bin/go"
    fi

    if [ -z "$GO_CMD" ] && [ -x "/usr/bin/go" ]; then
        GO_CMD="/usr/bin/go"
    fi

    if [ -z "$GO_CMD" ] && [ -x "/snap/bin/go" ]; then
        GO_CMD="/snap/bin/go"
    fi

    if [ -z "$GO_CMD" ] && [ -n "${SUDO_USER:-}" ]; then
        GO_CMD="$(su - "$SUDO_USER" -c 'command -v go' 2>/dev/null || true)"
    fi

    if [ -z "$GO_CMD" ]; then
        warn "Go não encontrado. Instalando via apt..."
        apt-get update -qq
        apt-get install -y golang-go
        GO_CMD="$(command -v go || true)"
    fi

    if [ -z "$GO_CMD" ]; then
        error "Go não encontrado após instalação."
    fi

    GO_VERSION=$("$GO_CMD" version | sed -n 's/.*go\([0-9][0-9]*\.[0-9][0-9]*\(\.[0-9][0-9]*\)\{0,1\}\).*/\1/p')
    if ! version_ge "$GO_VERSION" "$GO_MIN_VERSION"; then
        warn "Versão do Go ($GO_VERSION) pode ser antiga. Recomendado: >= $GO_MIN_VERSION"
        warn "Considere instalar uma versão mais recente via https://go.dev/dl/"
    else
        info "Go $GO_VERSION encontrado em $GO_CMD."
    fi
}

install_exiftool() {
    if command -v exiftool >/dev/null 2>&1; then
        info "ExifTool encontrado: $(exiftool -ver)"
        return
    fi

    info "ExifTool não encontrado. Instalando via apt..."
    apt-get update -qq
    apt-get install -y libimage-exiftool-perl

    if command -v exiftool >/dev/null 2>&1; then
        info "ExifTool instalado: $(exiftool -ver)"
    else
        error "Instalação do ExifTool falhou."
    fi
}

build() {
    info "Compilando $BINARY_NAME..."
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    cd "$SCRIPT_DIR"

    "$GO_CMD" mod download
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO_CMD" build -ldflags="-s -w" -o "$BINARY_NAME" .
    info "Build concluído."
}

install_binary() {
    info "Instalando $BINARY_NAME em $INSTALL_DIR..."
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    install -m 0755 "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    info "$BINARY_NAME instalado com sucesso em $INSTALL_DIR/$BINARY_NAME"
}

cleanup() {
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    rm -f "$SCRIPT_DIR/$BINARY_NAME"
}

verify() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        info "Verificação OK: $(command -v $BINARY_NAME)"
        "$BINARY_NAME" --help | head -5 || true
    else
        error "Instalação falhou: $BINARY_NAME não encontrado no PATH."
    fi
}

main() {
    echo "========================================="
    echo "  Instalador do $BINARY_NAME"
    echo "========================================="

    check_root
    check_go
    install_exiftool
    build
    install_binary
    cleanup
    verify

    echo ""
    info "Instalação concluída! Execute '$BINARY_NAME --help' para começar."
}

main "$@"
