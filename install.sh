#!/usr/bin/env bash
set -e

BINARY_NAME="mycli"
INSTALL_DIR="/usr/local/bin"
GO_MIN_VERSION="1.24.1"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()    { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "Este script precisa ser executado como root. Use: sudo ./install.sh"
    fi
}

version_ge() {
    # Retorna 0 se $1 >= $2
    printf '%s\n%s' "$2" "$1" | sort -C -V
}

check_go() {
    if ! command -v go &>/dev/null; then
        warn "Go não encontrado. Instalando via apt..."
        apt-get update -qq
        apt-get install -y golang-go
    fi

    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
    if ! version_ge "$GO_VERSION" "$GO_MIN_VERSION"; then
        warn "Versão do Go ($GO_VERSION) pode ser antiga. Recomendado: >= $GO_MIN_VERSION"
        warn "Considere instalar uma versão mais recente via https://go.dev/dl/"
    else
        info "Go $GO_VERSION encontrado."
    fi
}

install_exiftool() {
    if command -v exiftool &>/dev/null; then
        info "ExifTool encontrado: $(exiftool -ver)"
        return
    fi

    info "ExifTool não encontrado. Instalando via apt..."
    apt-get update -qq
    apt-get install -y libimage-exiftool-perl

    if command -v exiftool &>/dev/null; then
        info "ExifTool instalado: $(exiftool -ver)"
    else
        error "Instalação do ExifTool falhou."
    fi
}

build() {
    info "Compilando $BINARY_NAME..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cd "$SCRIPT_DIR"

    go mod download
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$BINARY_NAME" .
    info "Build concluído."
}

install_binary() {
    info "Instalando $BINARY_NAME em $INSTALL_DIR..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    install -m 0755 "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    info "$BINARY_NAME instalado com sucesso em $INSTALL_DIR/$BINARY_NAME"
}

cleanup() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    rm -f "$SCRIPT_DIR/$BINARY_NAME"
}

verify() {
    if command -v "$BINARY_NAME" &>/dev/null; then
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
