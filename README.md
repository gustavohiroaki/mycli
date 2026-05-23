# MyCLI - Conjunto de Ferramentas Pessoais

Uma CLI (Command Line Interface) pessoal que reúne várias ferramentas para aumentar a produtividade no desenvolvimento e automação de tarefas. Desenvolvida para ser um hub centralizado de utilitários personalizados.

## 🛠️ Ferramentas Disponíveis

### `prompt` - Refinador de Prompts
Refina e aprimora prompts usando a API da OpenAI para obter respostas mais precisas.

### `photo` - Workflow de Fotografia
Executa um menu guiado para ingestao de fotos e videos, com organizacao por metadados, estruturas de pastas customizaveis, tratamento de duplicados e relatorio final.

### `photo organize` - Organizacao direta de midia
Organiza fotos e videos por linha de comando, usando scan recursivo por padrao.

## ✨ Funcionalidades

- 🤖 **Integração com APIs externas** (OpenAI)
- 📝 **Interface interativa** para entrada de dados
- 📋 **Integração com clipboard** para facilitar o workflow
- 📄 **Suporte a arquivos de contexto**
- 📷 **Workflow de fotografia** com organizacao por data, camera, tipo de midia e tratamento de duplicados

## 🚀 Como usar

### Pré-requisitos

1. **Go 1.24.1** ou superior instalado
2. **Chave da API OpenAI** configurada como variável de ambiente para a ferramenta `prompt`
3. **ExifTool** recomendado para o workflow `photo`, usado para ler data real, camera e lente

### Configuração inicial

```bash
# Clone o repositório (se aplicável)
git clone <repository-url>
cd mycli

# Configure a chave da API da OpenAI
export OPENAI_API_KEY="sua-chave-da-api-aqui"
```

### Instalação Linux

```bash
# Execute o instalador
sudo ./install.sh
```

### Instalação Windows

No PowerShell, execute:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\install.ps1
```

O instalador compila o projeto, instala o executável em `%LOCALAPPDATA%\Programs\mycli` e adiciona esse diretório ao `PATH` do usuário.

### Execução manual

```bash
# Compile a aplicação manualmente
go build -o mycli

# Ou execute diretamente
go run main.go
```

### Comandos disponíveis

```bash
# Listar todas as ferramentas
./mycli help

# Usar a ferramenta de refinamento de prompts
./mycli prompt

# Usar com arquivo de contexto
./mycli prompt --context arquivo.txt

# Abrir o workflow guiado de fotografia
./mycli photo

# Organizar fotos e videos diretamente
./mycli photo organize ./entrada ./biblioteca
```

### Exemplo de uso - Ferramenta `prompt`

```bash
# 1. Execute o comando
./mycli prompt

# 2. Responda à pergunta interativa
# O que você quer fazer com o prompt? Criar um prompt para análise de dados

# 3. A ferramenta irá refinar o prompt e copiar para o clipboard
```

### Exemplo de uso - Workflow de Fotografia

```bash
./mycli photo
```

O menu guiado pergunta origem, destino, scan recursivo, exclusoes, modo copiar/mover, estrutura de pastas, renomeacao opcional, politica de duplicados e confirmacao antes de executar.

### Exemplo de uso - Organizacao direta

```bash
./mycli photo organize ./entrada ./biblioteca
./mycli photo organize ./entrada ./biblioteca --structure camera-date --duplicates skip
./mycli photo organize ./entrada ./biblioteca --no-recursive --exclude exports
```

O comando usa `exiftool` para ler data, camera e lente. Sem `exiftool`, o modo interativo pergunta se deve continuar com fallback limitado; o modo direto exige `--allow-fallback`.

Estruturas de pasta podem usar presets ou templates:

```bash
./mycli photo organize ./entrada ./biblioteca --structure "{camera}/{year}/{month}/{day}/{type}"
```

Tokens iniciais: `{year}`, `{month}`, `{day}`, `{date}`, `{time}`, `{camera}`, `{lens}`, `{type}`, `{extension}`.

## 🛠️ Desenvolvimento

### Comandos úteis

```bash
# Executar testes
go test ./...

# Formatar código
go fmt ./...

# Verificar problemas no código
go vet ./...

# Compilar para produção
go build -ldflags="-s -w" -o mycli
```

### Estrutura do projeto

```
mycli/
├── main.go              # Ponto de entrada da aplicação
├── cmd/
│   ├── root.go         # Comando raiz do Cobra
│   ├── prompt.go       # Comando principal de refinamento
│   ├── photo.go        # Workflow de fotografia
│   └── interactive.go  # Funções de interação com usuário
├── internal/photo/     # Motor de ingestao, metadados, templates e relatorios
├── go.mod              # Dependências do Go
└── README.md           # Este arquivo
```

## 📦 Dependências Principais

- **Cobra**: Framework para CLIs em Go
- **OpenAI Go Client**: Integração com APIs externas
- **Clipboard**: Manipulação da área de transferência
- **ExifTool**: Leitura de metadados de fotos, RAWs e videos para o workflow `photo`

## ⚙️ Configuração

### Variáveis de ambiente

| Variável | Descrição | Ferramenta |
|----------|-----------|------------|
| `OPENAI_API_KEY` | Chave da API da OpenAI | `prompt` |

### Arquivos de contexto

```bash
# Exemplo para a ferramenta prompt
echo "Contexto específico do projeto" > contexto.txt
./mycli prompt --context contexto.txt
```

## 🤝 Sobre

Este é meu conjunto de ferramentas pessoais desenvolvido para otimizar fluxos de trabalho e automatizar tarefas recorrentes. Cada ferramenta foi criada para resolver problemas específicos do meu dia a dia.

---

**Nota**: Desenvolvido para uso pessoal, mas pode ser adaptado para diferentes necessidades.
