# Exemplos do Comando `photo`

Este arquivo reúne exemplos práticos para o workflow de fotografia do `mycli`.

## Abrir o Menu Guiado

```bash
./mycli photo
```

Use quando quiser responder às perguntas no terminal: origem, destino, scan recursivo, exclusões, cópia ou movimento, estrutura de pastas, renomeação, duplicados e confirmação antes de executar.

## Organizar Fotos e Vídeos

```bash
./mycli photo organize ./entrada ./biblioteca
```

Por padrão, o comando:

- procura mídia recursivamente em `./entrada`;
- copia arquivos para `./biblioteca`;
- usa a estrutura `{year}/{month}/{day}/{type}`;
- mantém o nome original dos arquivos;
- gera `photo-ingest-report.txt`.

Exemplo de saída:

```text
./biblioteca/2026/05/24/photos/IMG_0001.JPG
./biblioteca/2026/05/24/raw/IMG_0001.CR3
./biblioteca/2026/05/24/videos/MVI_0001.MOV
./biblioteca/photo-ingest-report.txt
```

## Continuar Sem ExifTool

```bash
./mycli photo organize ./entrada ./biblioteca --allow-fallback
```

No modo direto, `exiftool` é exigido por padrão. Use `--allow-fallback` para continuar usando data no nome do arquivo ou data de modificação.

## Mover em Vez de Copiar

```bash
./mycli photo organize ./entrada ./biblioteca --move
```

Use com cuidado: os arquivos saem da origem depois de copiados para o destino.

## Desligar Scan Recursivo

```bash
./mycli photo organize ./entrada ./biblioteca --no-recursive
```

Processa apenas arquivos diretamente dentro de `./entrada`.

## Ignorar Pastas ou Padrões

```bash
./mycli photo organize ./entrada ./biblioteca --exclude exports --exclude thumbnails
```

Nada é ignorado automaticamente. O comando só ignora o que for passado com `--exclude`.

## Escolher Preset de Estrutura

```bash
./mycli photo organize ./entrada ./biblioteca --structure camera-date
```

Presets disponíveis:

```text
date         -> {year}/{month}/{day}/{type}
date-folder  -> {year}/{year}-{month}-{day}/{type}
camera-date  -> {camera}/{year}/{month}/{day}/{type}
year-camera  -> {year}/{camera}/{year}-{month}-{day}/{type}
legacy-pt    -> {year}/{month}/{day}/{type}
```

## Usar Estrutura Customizada

```bash
./mycli photo organize ./entrada ./biblioteca --structure "{year}/{camera}/{year}-{month}-{day}/{type}"
```

Tokens aceitos em templates:

```text
{year}
{month}
{day}
{date}
{time}
{camera}
{lens}
{type}
{extension}
```

## Renomear Arquivos

```bash
./mycli photo organize ./entrada ./biblioteca --rename "{date}_{time}_{camera}_{seq}{ext}"
```

Exemplo de nome gerado:

```text
2026-05-24_14-32-10_canon-eos-r6_001.jpg
```

Sem `--rename`, o nome original é preservado.

## Tratar Duplicados

Pular duplicados:

```bash
./mycli photo organize ./entrada ./biblioteca --duplicates skip
```

Separar duplicados:

```bash
./mycli photo organize ./entrada ./biblioteca --duplicates separate
```

Copiar duplicados com sufixo:

```bash
./mycli photo organize ./entrada ./biblioteca --duplicates suffix
```

Duplicados são detectados por hash SHA-256 do conteúdo. Nenhuma política apaga arquivos.

## Gerar Relatório JSON

```bash
./mycli photo organize ./entrada ./biblioteca --report json
```

Gera:

```text
photo-ingest-report.json
```

## Desligar Relatório

```bash
./mycli photo organize ./entrada ./biblioteca --report none
```

## Exemplo Completo

```bash
./mycli photo organize /media/cartao/DCIM ~/Fotos/Biblioteca \
  --structure "{camera}/{year}/{month}/{day}/{type}" \
  --rename "{date}_{time}_{camera}_{seq}{ext}" \
  --duplicates separate \
  --exclude exports \
  --report json
```

Esse exemplo organiza por câmera/data/tipo, renomeia arquivos, separa duplicados, ignora `exports` e gera relatório JSON.
