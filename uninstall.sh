#!/usr/bin/env bash
set -e

BINARY_NAME="mycli"
INSTALL_DIR="/usr/local/bin"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }

if [[ $EUID -ne 0 ]]; then
    error "Este script precisa ser executado como root. Use: sudo ./uninstall.sh"
fi

if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
    rm -f "$INSTALL_DIR/$BINARY_NAME"
    info "$BINARY_NAME removido de $INSTALL_DIR."
else
    info "$BINARY_NAME não estava instalado em $INSTALL_DIR."
fi
