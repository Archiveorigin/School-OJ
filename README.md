# 黄海在线题测平台 Monorepo

学校可部署 OJ 系统，固定目录：

- `apps/api`: Go 1.24 + Gin + GORM + PostgreSQL + Redis Streams + MinIO
- `apps/worker`: Go 1.24 judge-worker，Redis Streams consumer，Docker 沙箱
- `apps/web`: Vue3 + Vite + TypeScript + Element Plus + Monaco
- `deploy/compose`: Docker Compose 部署副本
- `deploy/k8s`: Kubernetes 参考清单
- `scripts`: 运维脚本
- `docs`: 架构、题目包、安全、部署文档

## 快速启动

```bash
docker compose up -d --build
```

首次判题前先拉取沙箱镜像：

```bash
./scripts/pull_sandbox_images.sh
docker compose restart worker
```

访问：

- Web: http://localhost:3000
- API: http://localhost:8080/healthz
- MinIO: http://localhost:9001
- Mailpit 邮件捕获: http://localhost:8025

本地启动会创建管理员、教师、学生种子账号用于功能验证；生产环境请替换初始凭据或关闭 `SEED_DATA`。

## 功能范围

- 学生/教师/管理员 RBAC
- 课程、班级、成员关系
- 题库与 ZIP 题目包，`problem.yaml` 校验
- C、C++、Python、Java 判题
- 作业、考试、排行榜
- SSE 实时提交状态
- JPlag 查重任务，支持 `JPLAG_JAR_PATH`，未配置时生成可测试的 fallback 报告
- 审计日志
- 用户 Profile、头像、邮箱换绑、反馈、账号注销
- 注册与邮箱找回密码，验证码邮件发件人为“黄海在线”
- Redis Streams judge-worker
- Docker 沙箱：禁网、只读根、tmpfs、非 root、`cap_drop=ALL`、`no-new-privileges`、seccomp、pids/cpu/memory/time/output limit

## 常用命令

```bash
make up
make logs
make smoke
make sandbox-images
make down
```

本地开发：

```bash
cd apps/api && go run ./cmd/api
cd apps/worker && go run ./cmd/worker
cd apps/web && npm install && npm run dev
```

测试：

```bash
make test
```

## 题目包

普通题目可以直接在 Web 页面创建：进入「题库」，点击「上传题目包」，切换到「表单创建题目」，填写题面、限制和测试点即可。系统会自动生成 `problem.yaml` 和 ZIP 包并存入 MinIO。

生成题目 ZIP：

```bash
./scripts/create_problem_zip.sh /tmp/a-plus-b.zip
```

教师或管理员可在 Web 的题库页面上传。格式见 [docs/problem-package.md](docs/problem-package.md)。

## 生产注意事项

`docker compose` 版本会挂载 Docker socket 给 worker，以便启动沙箱容器。生产环境建议将 worker 放在隔离节点，限制它只能访问专用 Docker daemon，并替换默认密钥、数据库密码和 MinIO 凭据。
