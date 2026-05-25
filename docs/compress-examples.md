# Exemplos do Comando `compress`

O comando `compress` compacta apenas videos usando FFmpeg. Fotos e RAWs sao ignorados.

## Compactar Um Video

```bash
./mycli compress ./lembranca.mov
```

Gera `lembranca_compressed.mp4` ao lado do original.

## Compactar Uma Pasta

```bash
./mycli compress ./videos --recursive --dest ./videos-comprimidos
```

Preserva subpastas relativas dentro do destino.

## Controlar Compressao

```bash
./mycli compress ./videos --recursive --level 35 --dest ./videos-comprimidos
```

Nivel:

```text
1   -> minima compressao, maior qualidade
35  -> default recomendado
100 -> maxima compressao, maior perda
```

## Substituir Originais

```bash
./mycli compress ./videos --recursive --replace
```

Com `--replace`, o original so e substituido se o video comprimido for valido e menor.
