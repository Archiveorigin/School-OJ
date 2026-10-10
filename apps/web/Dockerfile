FROM node:24-alpine AS build
WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci --registry=https://registry.npmmirror.com --no-audit --no-fund
COPY apps/web/ ./
COPY apps/api/openapi/openapi.json /app/apps/api/openapi/openapi.json
COPY apps/api/internal/handlers/routes.go /app/apps/api/internal/handlers/routes.go
RUN npm run build
RUN find dist -type f \( -name '*.js' -o -name '*.css' -o -name '*.svg' \) -exec gzip -9 -k {} \;

FROM nginx:stable-alpine@sha256:0985e772fb9f729e6fa0980da05fca5d9c468e870eed43071545afa9d2e27d94
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/web/dist /usr/share/nginx/html
EXPOSE 80
