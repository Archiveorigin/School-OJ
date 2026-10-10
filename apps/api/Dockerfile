FROM golang:1.26.9-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -o /out/api-healthcheck ./cmd/healthcheck

FROM eclipse-temurin:21-jre-alpine
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/api-healthcheck /usr/local/bin/api-healthcheck
EXPOSE 8080
ENTRYPOINT ["api"]
