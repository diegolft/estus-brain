# Deploy no Cubeship

O Estus Brain sobe como **dois apps** (`estus-backend` e `estus-frontend`)
mais um **datastore Postgres**. O Cubeship já roda Traefik na borda, então o
TLS e o certificado Let's Encrypt não são problema do projeto — não existe
mais proxy nem mTLS dentro do repo, e não deve existir: proxy dentro de proxy
só atrapalha.

A proteção do app agora é **login e senha**, não certificado de cliente.

## Desenho

```
Internet ──▶ Traefik (TLS, Let's Encrypt)
                │
                ▼
         estus-frontend  (Next.js, domínio estus.diegosalvador.com.br)
                │  rede interna, http://estus-backend:8080
                ▼
         estus-backend   (Go, SEM domínio público)
                │
                ▼
         datastore postgres
```

- **Só o frontend recebe domínio público.** Ele é a única coisa que o
  navegador toca.
- **O backend fica só na rede interna.** Não dê domínio a ele. Todo acesso do
  navegador ao Go já passa por Server Components, Server Actions e Route
  Handlers do Next — o Go nunca precisou ser público.
- **O Postgres é um datastore do Cubeship**, alcançável só pela rede interna.

## Ordem de criação

1. Crie o **datastore Postgres** primeiro e anote a connection string que o
   Cubeship gerar — é ela que vira `DATABASE_URL` do backend.
2. Crie o app **`estus-backend`** (build a partir de `backend/`, Dockerfile já
   no repo), com as envs da tabela abaixo. **Sem domínio público.**
3. Crie o app **`estus-frontend`** (build a partir de `frontend/`), com
   `API_URL` apontando para o nome interno do backend, e aí sim associe o
   domínio `estus.diegosalvador.com.br`.

O backend aplica as próprias migrations no boot; não há passo de migration
separado.

> O nome interno que o frontend usa em `API_URL` é o **nome do app backend**
> como você o criou no Cubeship. Se você chamou o app de `estus-backend`, a
> URL é `http://estus-backend:8080`. Se deu outro nome, ajuste.

## Envs — `estus-backend`

| Env | Obrigatória | Para quê |
|---|---|---|
| `DATABASE_URL` | **sim** | Connection string do datastore. O processo se recusa a subir sem ela. |
| `PORT` | não (8080) | Porta de escuta. Se o Cubeship injetar `PORT`, o app obedece. |
| `ESTUS_ADMIN_EMAIL` | **sim no 1º boot** | Cria o usuário inicial se não houver nenhum no banco. |
| `ESTUS_ADMIN_PASSWORD` | **sim no 1º boot** | Senha desse usuário. Depois do primeiro boot as duas podem sair. |
| `VAULT_ENCRYPTION_KEY` | não* | Chave mestra do cofre de senhas. Sem ela o módulo Senhas não monta; o resto do app funciona igual. *Obrigatória se você quer o cofre. |
| `FRONTEND_URL` | não | URL pública do frontend (`https://estus.diegosalvador.com.br`). Usada no redirect do OAuth do Google. |
| `DOCUMENTS_DIR` | não (`data/documents`) | Onde os arquivos enviados são gravados. Relativo ao WORKDIR `/app`. |
| `ASSISTANT_DIR` | não (`data/assistant`) | Estado local do assistente (token do MCP, scratch dirs). |
| `MCP_TOKEN` | não | Bearer token do `/mcp`. Vazio = gerado e salvo em `ASSISTANT_DIR`. |
| `ASSISTANT_TZ` | não (`America/Sao_Paulo`) | Fuso do dono: "hoje" e horários são lidos nele. |
| `ASSISTANT_MULTI_USER` | não (`false`) | `true` desliga os motores que usam assinatura pessoal de Claude/ChatGPT. |
| `TELEGRAM_BOT_TOKEN` | não | Bot do Telegram. Também dá para salvar pela tela. |
| `GOOGLE_CLIENT_ID` | não | Sincronização com Google Calendar (opcional). |
| `GOOGLE_CLIENT_SECRET` | não | Idem. |
| `GOOGLE_REDIRECT_URL` | não | Precisa bater exatamente com o registrado no Google Cloud Console. |

**Volume persistente:** monte um volume em `/app/data` se você usa o módulo
Documentos. Sem isso os arquivos enviados somem no próximo deploy — eles
ficam em disco, só os metadados vão pro Postgres.

## Envs — `estus-frontend`

| Env | Obrigatória | Para quê |
|---|---|---|
| `API_URL` | **sim** | Endereço interno do backend: `http://estus-backend:8080`. É por aqui que o Next fala com o Go. |
| `PORT` | não (3000) | Porta de escuta. O `next start` lê `PORT` nativamente. |

E o domínio `estus.diegosalvador.com.br` apontado para este app, com o DNS do
domínio resolvendo para o IP da VPS.

## Gerando a `VAULT_ENCRYPTION_KEY`

```bash
openssl rand -base64 32
```

Cole o resultado na env `VAULT_ENCRYPTION_KEY` do **backend**.

Dois avisos que valem o incômodo:

- **Se você perder essa chave, perde as senhas do cofre.** Elas estão
  cifradas em AES-256-GCM no banco e a chave não está lá junto.
- **Se você trocar a chave**, as senhas já salvas não descriptografam mais.
  Não é uma rotação suave.
- Guarde-a **fora** do backup do Postgres. Backup do banco sem a chave é
  ruído para quem não deveria ler; junte os dois no mesmo lugar e a proteção
  virou enfeite.

## Primeiro acesso

1. Suba o backend com `ESTUS_ADMIN_EMAIL` e `ESTUS_ADMIN_PASSWORD` definidas.
2. No boot, sem nenhum usuário no banco, ele cria esse primeiro usuário.
3. Abra `https://estus.diegosalvador.com.br` e entre com esse e-mail e senha.
4. Depois disso as duas envs podem ser removidas do app — elas só agem quando
   o banco não tem usuário nenhum.

## Antes de considerar "no ar"

- **Backup do Postgres.** `pg_dump` agendado, com a
  `VAULT_ENCRYPTION_KEY` guardada em outro lugar.
- **O login é single-user.** Existe autenticação de verdade, mas as tabelas
  de dados não têm `user_id`: um segundo usuário criado no banco enxerga
  exatamente os mesmos dados do primeiro. Isso é login, não multiusuário —
  não convide ninguém achando que os dados ficam separados.
