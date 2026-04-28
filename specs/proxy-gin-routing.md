# Especificação Simplificada - Proxy com Gin

## Objetivo

Construir um proxy HTTP em Go usando Gin, com configuração externa em JSON para definir regras de roteamento (redirect/proxy pass-through) de forma simples.

## Requisitos Funcionais

### 1) Configuração por JSON

O proxy deve carregar um arquivo JSON contendo as regras de roteamento.

Cada regra deve permitir:

- identificar a requisição por `método HTTP + path`;
- suportar `path` exato e com coringa (`*`);
- opcionalmente redirecionar com base em header configurável (`header_name` e `header_value` opcionais).

### 2) Match por método + path (formato golang)

O formato da chave de rota deve seguir:

- `<VERBO_HTTP> <PATH>`

Exemplos válidos:

- `PATCH /receivables`
- `PATCH /receivables/*`
- `GET /health`

### 3) Suporte a coringas

Deve ser possível configurar padrões com `*` para capturar subpaths.

Exemplo:

- `PATCH /receivables/*` deve casar com:
  - `PATCH /receivables/123`
  - `PATCH /receivables/abc/items`

### 4) Redirect baseado em Header

Deve ser possível configurar uma regra que avalia um header da requisição e seleciona um destino de redirect/proxy com base no valor.

Cenário solicitado:

- Header configurável: `X-Header-Redirect`
- Se valor for `journey-x`, deve apontar para a rota/destino `X` (definido em configuração).

## Modelo de Configuração (JSON)

Exemplo mínimo de configuração:

```json
{
  "routes": [
    {
      "match": "PATCH /receivables",
      "target": "http://service-a.internal"
    },
    {
      "match": "PATCH /receivables/*",
      "target": "http://service-b.internal"
    },
    {
      "match": "PATCH /receivables",
      "header_name": "X-Header-Redirect",
      "header_value": "journey-x",
      "target": "http://service-x.internal"
    }
  ]
}
```

## Regras de Decisão (ordem de prioridade)

Para simplificar e evitar ambiguidades:

1. Considerar apenas regras de `routes` cujo `match` case com `método + path`.
2. Entre as regras candidatas, priorizar primeiro as que possuem `header_name` e `header_value` definidos.
3. Para regras com header, o match só é válido quando:
   - o header `header_name` existir na requisição; e
   - o valor for igual a `header_value`.
4. Se nenhuma regra com header casar, usar regra sem header (fallback).
5. Em empate, priorizar:
   - primeiro path exato;
   - depois path com coringa;
   - por fim, a primeira regra declarada no JSON.
6. Se nada casar, retornar `404`.

## Comportamento Esperado (cenários)

### Cenário A - Match exato

- Entrada: `PATCH /receivables`
- Resultado: encaminhar para target da regra exata.

### Cenário B - Match com coringa

- Entrada: `PATCH /receivables/123`
- Resultado: encaminhar para target da regra `PATCH /receivables/*`.

### Cenário C - Header define destino

- Entrada: `PATCH /receivables` com `X-Header-Redirect: journey-x`
- Resultado: encaminhar para o `target` da regra em `routes` que contém `header_name` + `header_value`.

### Cenário D - Header sem match

- Entrada: `PATCH /receivables` com `X-Header-Redirect: unknown`
- Resultado: ignorar a regra com header e seguir para regra sem header em `routes`.

### Cenário E - Sem regra

- Entrada: qualquer requisição sem match
- Resultado: `404 Not Found`.

## Critérios de Aceite

- O proxy carrega o JSON na inicialização sem erro.
- Regras no formato `VERBO + espaço + PATH` são aceitas.
- Paths com `*` funcionam para subpaths.
- Header configurável influencia o destino quando presente e mapeado.
- A ordem de prioridade definida nesta especificação é respeitada.
