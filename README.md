# proxy

Proxy HTTP simples em Go com Gin, configurado por arquivo JSON.

## Requisitos

- Go 1.26+

## Configuração

Crie um arquivo `config.json` na raiz com regras em `routes`.

Exemplo:

```json
{
  "routes": [
    {
      "match": "PATCH /receivables",
      "rewrite": "PUT /v2/receivables",
      "target": "http://localhost:9001"
    },
    {
      "match": "PATCH /receivables/*",
      "rewrite": "PUT /v2/receivables/*",
      "target": "http://localhost:9002"
    },
    {
      "match": "PATCH /receivables",
      "header_name": "X-Header-Redirect",
      "header_value": "journey-x",
      "rewrite": "POST /v3/receivables",
      "target": "http://localhost:9003"
    }
  ]
}
```

Campos por rota:

- `match`: formato `<METHOD> <PATH>` (ex: `PATCH /receivables`).
- `target`: URL base de destino.
- `header_name` (opcional): nome do header para filtro.
- `header_value` (opcional): valor esperado do header.
- `rewrite` (opcional): formato `<METHOD> <PATH>` para reescrever método e path antes do forward.

Observação: `header_name` e `header_value` devem ser informados juntos.
Observação: para regras com `/*`, você pode usar `rewrite` com `/*` para preservar o sufixo (ex: `/receivables/123` -> `/v2/receivables/123`).

## Como rodar

```bash
go run . -config config.json -listen :8080
```

## Como funciona a prioridade

Para o mesmo `match`:

1. Regras com `header_name` + `header_value` válidos na requisição.
2. Regras sem header (fallback).
3. Em empate: path exato > path com `/*` > ordem declarada no JSON.

Sem match, a resposta é `404`.

## Testes

```bash
go test ./...
```