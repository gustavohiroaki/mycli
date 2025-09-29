# MyCLI - Conjunto de Ferramentas Pessoais

Uma CLI (Command Line Interface) pessoal que reúne várias ferramentas para aumentar a produtividade no desenvolvimento e automação de tarefas. Desenvolvida para ser um hub centralizado de utilitários personalizados.

## 🛠️ Ferramentas Disponíveis

### `prompt` - Refinador de Prompts
Refina e aprimora prompts usando a API da OpenAI para obter respostas mais precisas.

## ✨ Funcionalidades

- 🤖 **Integração com APIs externas** (OpenAI)
- 📝 **Interface interativa** para entrada de dados
- 📋 **Integração com clipboard** para facilitar o workflow
- 📄 **Suporte a arquivos de contexto**

## 🚀 Como usar

### Pré-requisitos

1. **Go 1.24.1** ou superior instalado
2. **Chave da API OpenAI** configurada como variável de ambiente

### Configuração inicial

```bash
# Clone o repositório (se aplicável)
git clone <repository-url>
cd mycli

# Configure a chave da API da OpenAI
export OPENAI_API_KEY="sua-chave-da-api-aqui"
```

### Instalação

```bash
# Compile a aplicação
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
```

### Exemplo de uso - Ferramenta `prompt`

```bash
# 1. Execute o comando
./mycli prompt

# 2. Responda à pergunta interativa
# O que você quer fazer com o prompt? Criar um prompt para análise de dados

# 3. A ferramenta irá refinar o prompt e copiar para o clipboard
```

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
│   └── interactive.go  # Funções de interação com usuário
├── go.mod              # Dependências do Go
└── README.md           # Este arquivo
```

## 📦 Dependências Principais

- **Cobra**: Framework para CLIs em Go
- **OpenAI Go Client**: Integração com APIs externas
- **Clipboard**: Manipulação da área de transferência

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