# Exemplos de `curl` para testar redirects

Assumindo que o proxy está rodando em `http://localhost:8080` com o `config.json` atual.

## 1) Match exato sem header (`PATCH /receivables`)

Deve redirecionar para o target `http://localhost:8000` com rewrite para
`PUT /v2/receivables`.

```bash
curl -i -X PATCH "http://localhost:8080/receivables" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000}'
```

## 2) Match wildcard (`PATCH /receivables/*`)

Deve redirecionar para o target `http://localhost:8000` com rewrite para
`PUT /v2/receivables/*`, preservando o sufixo.

```bash
curl -i -X PATCH "http://localhost:8080/receivables/123" \
  -H "Content-Type: application/json" \
  -d '{"status":"updated"}'
```

## 3) Prioridade de header (`X-Header-Redirect: journey-x`)

Mesmo path exato (`/receivables`), com esse header a regra de header deve ter prioridade
e redirecionar para `http://localhost:8000` com rewrite para `POST /v3/receivables`.

```bash
curl -i -X PATCH "http://localhost:8080/receivables" \
  -H "X-Header-Redirect: journey-x" \
  -H "Content-Type: application/json" \
  -d '{"route":"header-priority"}'
```

## 4) Header inválido faz fallback para regra sem header

Com valor diferente de `journey-x`, deve cair na regra exata sem header
e redirecionar para `http://localhost:8000` com rewrite para `PUT /v2/receivables`.

```bash
curl -i -X PATCH "http://localhost:8080/receivables" \
  -H "X-Header-Redirect: outro-valor" \
  -H "Content-Type: application/json" \
  -d '{"route":"fallback"}'
```

## 5) Sem regra: deve retornar `404`

```bash
curl -i -X GET "http://localhost:8080/does-not-exist"
```

## 6) Wildcard DELETE (`DELETE /receivables/*`)

Deve redirecionar para o target `http://localhost:8000` com rewrite para
`DELETE /v2/receivables/*`, preservando o sufixo do path.

```bash
curl -i -X DELETE "http://localhost:8080/receivables/abc-123"
```

## 7) Wildcard DELETE (`DELETE /receivables/*/product_items`)

Deve redirecionar para o target `http://localhost:8000` com rewrite para
`DELETE /v2/receivables/*`, preservando o sufixo do path.

```bash
curl -i -X DELETE "http://localhost:8080/receivables/abc-123/items"
```