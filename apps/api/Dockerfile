FROM node:24-alpine AS ranking-export
WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci --registry=https://registry.npmmirror.com --no-audit --no-fund
COPY apps/web/ ./
RUN npm run build:ranking-export

FROM golang:1.26.9-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -o /out/api-healthcheck ./cmd/healthcheck

FROM eclipse-temurin:21-jre-alpine
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/api-healthcheck /usr/local/bin/api-healthcheck
COPY --from=ranking-export /app/apps/web/dist/offline-ranking.html /usr/local/share/school-oj/offline-ranking.html
EXPOSE 8080
ENTRYPOINT ["api"]
