# Build context: frontend/ (é assim que o docker-compose chama: `build: ./frontend`).
# Todos os COPY abaixo são relativos a frontend/, não à raiz do repo.
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# `prebuild` (scripts/copy-excalidraw-assets.mjs) gera public/excalidraw-assets/
# a partir do node_modules antes do `next build`.
RUN npm run build

FROM node:20-alpine
RUN addgroup -S estus && adduser -S estus -G estus
WORKDIR /app
ENV NODE_ENV=production
# O projeto NÃO usa `output: "standalone"` no next.config.ts, então o runtime
# é o `next start` clássico: ele precisa de .next, public, node_modules,
# package.json e do próprio next.config.ts em disco.
COPY --from=build /app/public ./public
COPY --from=build /app/.next ./.next
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/package.json ./package.json
COPY --from=build /app/next.config.ts ./next.config.ts
USER estus
# Documental apenas. A porta real vem da env PORT, que o `next start` lê
# nativamente (cai em 3000 se não definida) — é assim que o Cubeship injeta
# a porta do container.
EXPOSE 3000
CMD ["npm", "start"]
