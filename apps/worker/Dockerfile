FROM golang:1.24-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o /out/worker ./cmd/worker

FROM docker:28-cli
COPY --from=build /out/worker /usr/local/bin/worker
COPY seccomp/oj-seccomp.json /etc/oj-seccomp.json
ENTRYPOINT ["worker"]
