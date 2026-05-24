# MyCLI - Conjunto de Ferramentas Pessoais

Uma CLI (Command Line Interface) pessoal que reúne várias ferramentas para aumentar a produtividade no desenvolvimento e automação de tarefas. Desenvolvida para ser um hub centralizado de utilitários personalizados.

## 🛠️ Ferramentas Disponíveis

### `prompt` - Refinador de Prompts
Refina e aprimora prompts usando a API da OpenAI para obter respostas mais precisas.

### `photo` - Workflow de Fotografia
Executa um menu guiado ou organiza direto quando recebe origem e destino, com organizacao por metadados, estruturas de pastas customizaveis, tratamento de duplicados, agrupamento por burst/similaridade e relatorio final.

### `photo organize` - Organizacao direta de midia
Organiza fotos e videos por linha de comando, usando scan recursivo por padrao.

### `unpack` - Descompactador seguro em lote
Descompacta arquivos `.zip`, `.tar`, `.tar.gz` e `.tgz`, verifica os arquivos extraidos e apaga cada compactado original somente depois de uma verificacao bem-sucedida.

## ✨ Funcionalidades

- 🤖 **Integração com APIs externas** (OpenAI)
- 📝 **Interface interativa** para entrada de dados
- 📋 **Integração com clipboard** para facilitar o workflow
- 📄 **Suporte a arquivos de contexto**
- 📷 **Workflow de fotografia** com organizacao por data, camera, tipo de midia, tratamento de duplicados e agrupamento por burst/similaridade
- 📦 **Descompactacao segura** em lote com verificacao antes de apagar originais

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

# Abrir o workflow guiado com origem e destino preenchidos
./mycli photo ./entrada ./biblioteca

# Organizar fotos e videos diretamente, sem perguntas
./mycli photo organize ./entrada ./biblioteca

# Descompactar um arquivo e apagar o original depois da verificacao
./mycli unpack arquivo.zip

# Descompactar compactados de uma pasta
./mycli unpack ./downloads
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

### Exemplo de uso - Workflow guiado com caminhos

```bash
./mycli photo ./entrada ./biblioteca
```

Esse modo usa os caminhos informados e ainda pergunta as demais opcoes antes de executar.

### Exemplo de uso - Organizacao direta

```bash
./mycli photo organize ./entrada ./biblioteca
./mycli photo organize ./entrada ./biblioteca --structure camera-date --duplicates skip
./mycli photo organize ./entrada ./biblioteca --no-recursive --exclude exports
./mycli photo organize ./entrada ./biblioteca --rename grouped --burst-window 2s --similarity-threshold 8
./mycli photo organize ./entrada ./biblioteca --fullperformance
```

O comando usa `exiftool` para ler data, camera e lente. Sem `exiftool`, o modo interativo pergunta se deve continuar com fallback limitado; o modo direto exige `--allow-fallback`.

Estruturas de pasta podem usar presets ou templates:

```bash
./mycli photo organize ./entrada ./biblioteca --structure "{camera}/{year}/{month}/{day}/{type}"
```

Tokens iniciais: `{year}`, `{month}`, `{day}`, `{date}`, `{time}`, `{camera}`, `{lens}`, `{type}`, `{extension}`.

Use `--rename grouped` para dar nomes parecidos a fotos relacionadas. Combine com `--burst-window 2s` para agrupar sequencias por tempo e com `--similarity-threshold 8` para agrupar imagens visualmente parecidas quando o formato puder ser decodificado. Esses grupos afetam nomes e relatorios; nao criam pastas extras nem apagam fotos similares.

Use `--fullperformance` para executar leitura de metadados, hashes e copia/movimento em paralelo usando os CPUs disponiveis. A ordem dos logs pode refletir a ordem de conclusao dos arquivos.

Veja mais cenários em [docs/photo-examples.md](docs/photo-examples.md).

### Exemplo de uso - Descompactador seguro

```bash
# Descompactar um arquivo e apagar o original depois da verificacao
./mycli unpack arquivo.zip

# Descompactar compactados de uma pasta
./mycli unpack ./downloads

# Procurar tambem em subpastas
./mycli unpack ./downloads --recursive

# Preservar os originais
./mycli unpack ./downloads --keep

# Extrair em outro destino
./mycli unpack ./downloads --dest ./extraidos
```

O `unpack` apaga cada arquivo compactado automaticamente apenas depois de extrair e verificar aquele arquivo. Use `--keep` para preservar os compactados originais.

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
│   ├── unpack.go       # Descompactador seguro
│   └── interactive.go  # Funções de interação com usuário
├── docs/
│   └── photo-examples.md # Exemplos de uso do comando photo
├── internal/archive/   # Descoberta, extracao e verificacao de compactados
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
