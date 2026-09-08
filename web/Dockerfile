FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install

FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# Адреса API, слаг арендатора и бренд в сборку НЕ вшиваются: образ один на всех
# клиентов, значения приходят в рантайме через /api/config.js (lib/runtime-config.ts).
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ARG VORTANIX_VERSION=dev
ENV VORTANIX_VERSION=$VORTANIX_VERSION
# Апдейтер читает версию работающего контейнера из этого label, не поднимая его.
LABEL org.vortanix.version=$VORTANIX_VERSION
LABEL org.vortanix.component=panel-ui
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
ENV PORT=3000
# Без этого Next в standalone слушает адрес, на который указывает HOSTNAME,
# а docker кладёт туда идентификатор контейнера: снаружи панель отвечает,
# а изнутри 127.0.0.1 отказывает — и служба обновления считала исправную
# версию нерабочей и откатывала её.
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
