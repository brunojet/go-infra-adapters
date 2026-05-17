# Exemplo: cmd/example

Este pequeno exemplo demonstra como chamar as APIs de `Secret` e `Storage` expostas
pelos providers registrados no runtime.

Pré-requisitos
- Credenciais AWS configuradas no ambiente (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`), ou um endpoint compatível (ex: LocalStack) configurado via `AWS_ENDPOINT_URL`.

Como executar

Executar diretamente (usa o `init()` dos providers registrados):

```bash
go run ./cmd/example
```

Compilar e executar binário:

```bash
go build -o example ./cmd/example
./example
```

Observações
- O exemplo tenta obter o provider `aws` via registry; se o provider não expor
  `Storage`, uma mensagem será mostrada e a parte de storage será pulada.
- O exemplo demonstra `PutObject` com `ContentType`, `HeadObject` preenchendo
  um `ObjectInfo` e `GetObject` preenchendo um `BucketObject`. Veja também o
  changelog em [CHANGELOG.md](../../CHANGELOG.md) para notas sobre mudanças
  incompatíveis na API de storage.
