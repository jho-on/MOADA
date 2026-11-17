# MOADA

Serviço próprio de armazenamento em nuvem com criptografia ponta a ponta (E2EE), interface web e API pública.

## Stack

- **Backend:** Go
- **Frontend:** Templ + HTMX + JavaScript
- **Banco de dados:** PostgreSQL
- **Object storage:** Garage (API compatível com S3)
- **API pública:** REST versionada em `/api/v1/`

## Estrutura do repositório

```text
cmd/
  moada/                 # binário da aplicação web
  migrate/               # binário de migrations
internal/
  config/                # leitura e validação de variáveis de ambiente
  database/              # pool PostgreSQL e execução de migrations
  storage/               # cliente S3 do Garage (apenas bytes criptografados)
  health/                # verificação de dependências
  httpapi/               # rotas, middlewares e handlers HTTP
  web/
    templates/           # componentes Templ
    static/              # CSS, JavaScript e dependências vendorizadas
migrations/              # migrations SQL embutidas no binário
deploy/
  docker-compose.yml     # PostgreSQL + Garage para desenvolvimento
  garage/garage.toml     # configuração local do Garage
```

## Pré-requisitos

- Go 1.26 ou superior
- Docker com Docker Compose
- `make` (opcional, mas os exemplos abaixo usam os atalhos)

## Desenvolvimento local

1. Crie o arquivo de ambiente:

   ```sh
   make env
   ```

   Os valores padrão em `.env.example` correspondem à infraestrutura local e são **apenas para desenvolvimento**.

2. Suba PostgreSQL e Garage:

   ```sh
   make dev-up
   ```

   O Compose aguarda os serviços ficarem saudáveis. O Garage cria automaticamente o nó único, a chave de acesso e o bucket configurados.

3. Aplique as migrations:

   ```sh
   make migrate-up
   ```

4. Inicie a aplicação:

   ```sh
   make run
   ```

5. Acesse <http://localhost:8080>.

### Serviços locais

| Serviço | Endereço local |
|---|---|
| Aplicação web | `http://localhost:8080` |
| PostgreSQL | `localhost:5434` |
| Garage S3 | `http://localhost:3900` |
| Garage Admin API | `http://localhost:3903` |


## Configuração

Toda a configuração vem de variáveis de ambiente. O `Makefile` carrega o arquivo `.env` automaticamente quando ele existe.

| Variável | Padrão | Descrição |
|---|---|---|
| `MOADA_ENV` | `development` | `development`, `test` ou `production`. Em produção os logs saem em JSON e HSTS é habilitado. |
| `MOADA_HTTP_ADDR` | `:8080` | Endereço de escuta HTTP. |
| `MOADA_LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error`. |
| `MOADA_DATABASE_URL` | — | URL PostgreSQL (`postgres://...`). Obrigatória. |
| `MOADA_DB_MAX_CONNS` | `10` | Tamanho máximo do pool de conexões. |
| `MOADA_DB_CONNECT_TIMEOUT` | `10s` | Timeout da conexão inicial com o banco. |
| `MOADA_S3_ENDPOINT` | — | Host e porta do Garage, sem esquema. Obrigatória. |
| `MOADA_S3_ACCESS_KEY` | — | Access key S3. Obrigatória. |
| `MOADA_S3_SECRET_KEY` | — | Secret key S3. Obrigatória. |
| `MOADA_S3_BUCKET` | — | Bucket de objetos. Obrigatório. |
| `MOADA_S3_REGION` | `garage` | Região S3 esperada pelo Garage. |
| `MOADA_S3_USE_SSL` | `false` | Use TLS ao conectar ao Garage. |
| `MOADA_S3_CREATE_BUCKET` | `true` | Cria o bucket na inicialização se ele não existir. |

Nunca comite o arquivo `.env` nem credenciais reais. As credenciais e tokens presentes em `deploy/` e `.env.example` são exclusivamente de desenvolvimento.

## Migrations

As migrations ficam em `migrations/` e são embutidas no binário via `embed`. O controle de versão é feito pelo Goose.

```sh
make migrate-up       # aplica todas as migrations pendentes
make migrate-down     # reverte uma migration
make migrate-status   # mostra o estado atual
```

Em produção, execute o binário `bin/moada-migrate up` antes de iniciar a aplicação.

## Testes

```sh
make test               # testes unitários
make vet                # análise estática
make check              # formata, analisa e testa
```

Testes de integração usam a infraestrutura local e são pulados por padrão:

```sh
make dev-up
make test-integration
```

Eles cobrem upload/leitura/remoção no Garage e a aplicação das migrations no PostgreSQL. Use um banco descartável: as migrations rodam no banco indicado por `MOADA_DATABASE_URL`.

## Endpoints

| Método | Caminho | Descrição |
|---|---|---|
| `GET` | `/` | Página inicial renderizada com Templ. |
| `GET` | `/partials/status` | Fragmento HTMX com o status dos serviços. |
| `GET` | `/healthz` | Liveness probe (não verifica dependências). |
| `GET` | `/readyz` | Readiness probe (PostgreSQL + Garage). Retorna `503` se algo falhar. |
| `GET` | `/static/*` | CSS, JavaScript e dependências. |

