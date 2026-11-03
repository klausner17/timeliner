# Build do app Vue e servidor nginx com o dist e o proxy de /api.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# Subcaminho de producao (ex.: /app/timeliner/); o padrao serve na raiz.
ARG VITE_BASE=/
RUN VITE_BASE=$VITE_BASE npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY security-headers.inc /etc/nginx/conf.d/security-headers.inc
COPY --from=build /app/dist /usr/share/nginx/html
ENV API_UPSTREAM=http://backend:8080
# Rede do proxy da frente, se houver (ex.: 172.16.0.0/12 para um proxy no host do Docker).
ENV REAL_IP_FROM=127.0.0.1
EXPOSE 80
