# Changelog

## [3.1.0](https://github.com/brunojet/go-infra-adapters/compare/v3.0.0...v3.1.0) (2026-05-30)


### Features

* add mockgen mocks for public contract interfaces ([8d28f15](https://github.com/brunojet/go-infra-adapters/commit/8d28f15c8a6b66833a886206bf1df16d28b14415))
* add mockgen-generated mocks for all public contract interfaces ([97c2ba3](https://github.com/brunojet/go-infra-adapters/commit/97c2ba301653a63cdd58b1a9dafd041a0d46cfc4))

## [3.0.0](https://github.com/brunojet/go-infra-adapters/compare/v2.0.0...v3.0.0) (2026-05-30)


### ⚠ BREAKING CHANGES

* pkg/cloudfront/signer → pkg/crypto; pkg/provider removed

### Features

* expose HealthCheck in public SecretAdapter interface ([9c5f7e1](https://github.com/brunojet/go-infra-adapters/commit/9c5f7e1c7361f8ca74d7225b8ce01a51c507faed))
* expose HealthCheck in public SecretAdapter interface ([b6544a6](https://github.com/brunojet/go-infra-adapters/commit/b6544a66bcd9b2223d5f8805d24e7bad1de25059))


### Code Refactoring

* consolidate adapters, replace signer with crypto, simplify … ([4c8ce40](https://github.com/brunojet/go-infra-adapters/commit/4c8ce40ced09384c2a6fac793f93f0d19b9eca95))
* consolidate adapters, replace signer with crypto, simplify provider ([440272c](https://github.com/brunojet/go-infra-adapters/commit/440272cb66e217a5dd59ed48a1ce2606d2e6c1dd))

## [2.0.0](https://github.com/brunojet/go-infra-adapters/compare/v1.0.0...v2.0.0) (2026-05-30)


### ⚠ BREAKING CHANGES

* pkg/cloudfront/signer → pkg/crypto; pkg/provider removed

### Code Refactoring

* consolidate adapters, replace signer with crypto, simplify … ([4c8ce40](https://github.com/brunojet/go-infra-adapters/commit/4c8ce40ced09384c2a6fac793f93f0d19b9eca95))
* consolidate adapters, replace signer with crypto, simplify provider ([440272c](https://github.com/brunojet/go-infra-adapters/commit/440272cb66e217a5dd59ed48a1ce2606d2e6c1dd))

## CHANGELOG

## Unreleased

### Breaking changes (storage)

- Introduzido `BucketObject` em `pkg/storage/contracts` para unificar stream + metadata.
  - `GetObject(ctx, key, obj *BucketObject) error` — agora preenche o `obj` fornecido com `Body` e `Info` (chamar `obj.Close()` quando terminar).
  - `PutObject(ctx, obj *BucketObject) error` — aceita `BucketObject` para permitir enviar `ContentType` e `Metadata` junto com o `Body`.
  - `HeadObject(ctx, key, objInfo *ObjectInfo) error` — preenche o `objInfo` fornecido com metadata sem transferir o corpo.
  - `ObjectInfo` agora contém `ContentType string`.

- Observações de migração:
  - Atualize mocks gerados (gomock) para as novas assinaturas.
  - Chamadores de `GetObject` devem alocar `&BucketObject{}` e chamar `Close()` após leitura.
  - A implementação `PutObject` pode fechar `obj.Body` após o upload; não dependa de `obj.Body` permanecer aberto após a chamada.

### Suggested PR title

`storage: migrate to BucketObject-based streaming API (breaking)`

### Migration snippet

Antes:

```go
rc, info, err := bkt.GetObject(ctx, "key")
defer rc.Close()
```

Depois:

```go
obj := &storagecontracts.BucketObject{}
if err := bkt.GetObject(ctx, "key", obj); err != nil { /* ... */ }
defer obj.Close()
```
