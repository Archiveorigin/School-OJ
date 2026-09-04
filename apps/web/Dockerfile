FROM node:24-alpine AS build
WORKDIR /app/apps/web
COPY apps/web/package.json ./
RUN npm config set registry https://registry.npmmirror.com && npm install
COPY apps/web/ ./
COPY apps/api/openapi/openapi.json /app/apps/api/openapi/openapi.json
COPY apps/api/internal/handlers/routes.go /app/apps/api/internal/handlers/routes.go
RUN npm run build

FROM nginx:1.27-alpine
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/web/dist /usr/share/nginx/html
EXPOSE 80
