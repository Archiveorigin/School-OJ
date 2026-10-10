FROM golang:1.26.9-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -o /out/worker ./cmd/worker

FROM build AS integration-build
RUN CGO_ENABLED=0 GOOS=linux go test -c -o /out/runner-integration ./internal/runner

FROM docker:28-cli AS integration
COPY --from=integration-build /out/runner-integration /usr/local/bin/runner-integration
ENTRYPOINT ["runner-integration", "-test.run=TestDockerSandboxMetricsIntegration", "-test.v"]

FROM docker:28-cli
COPY --from=build /out/worker /usr/local/bin/worker
COPY seccomp/oj-seccomp.json /etc/oj-seccomp.json
ENTRYPOINT ["worker"]
