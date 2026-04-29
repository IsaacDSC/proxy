# proxy

Proxy HTTP simples em Go com Gin, configurado por arquivo JSON.

## Requisitos

- Go 1.26+

## Configuração

Crie um arquivo `config.json` na raiz com regras em `routes`.

Exemplo:

```json
{
    "transport": {
        "dial_timeout": "30s",
        "keep_alive": "30s",
        "max_idle_conns": 100,
        "max_conns_per_host": 100,
        "idle_conn_timeout": "90s",
        "expect_continue_timeout": "1s"
    },
    "routes": [
        {
            "match": "DELETE /receivables/*/items",
            "rewrite": "DELETE /v2/receivables/*/product_items",
            "target": "http://localhost:8000"
        },
        {
            "match": "DELETE /receivables/*",
            "rewrite": "DELETE /v2/receivables/*",
            "target": "http://localhost:8000"
        },
        {
            "match": "PATCH /receivables",
            "rewrite": "PUT /v2/receivables",
            "target": "http://localhost:8000"
        },
        {
            "match": "PATCH /receivables/*",
            "rewrite": "PUT /v2/receivables/*",
            "target": "http://localhost:8000"
        },
        {
            "match": "PATCH /receivables",
            "header_name": "X-Header-Redirect",
            "header_value": "journey-x",
            "rewrite": "POST /v3/receivables",
            "target": "http://localhost:8000",
            "transport": {
                "dial_timeout": "5s",
                "keep_alive": "15s",
                "max_idle_conns": 10,
                "max_conns_per_host": 10,
                "idle_conn_timeout": "30s",
                "expect_continue_timeout": "500ms"
            }
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