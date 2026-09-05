# envdrift

**在部署前发现 `.env` 配置漂移。**

`envdrift` 是一个零依赖的 Go 命令行工具。它将 `.env.example` 视为配置契约，并检查所有目标环境文件是否与之保持一致：缺少变量、未声明变量、重复定义，以及（可选的）变量值差异都会被发现。

> 不要把部署到预发布或生产环境，当成配置是否完整的第一次测试。

## 它解决什么问题

一个项目通常不只拥有一个 dotenv 文件：`.env.example`、`.env.local`、CI、预发布和生产环境的配置会在迭代中慢慢分叉。比如新增了 `STRIPE_WEBHOOK_SECRET`，但生产环境没有同步，往往要到最不方便的时候才暴露问题。

`envdrift` 让这份配置契约可以在本地、Git hook 或 CI 中自动验证。

## 安装

```sh
go install github.com/qvoo/envdrift/cmd/envdrift@latest
```

也可以直接运行，无须安装：

```sh
go run github.com/qvoo/envdrift/cmd/envdrift@latest .env.example .env.local
```

## 快速开始

```dotenv
# .env.example
PORT=8080
DATABASE_URL=
STRIPE_WEBHOOK_SECRET=
```

```dotenv
# .env.staging
PORT=8080
DATABASE_URL=postgres://…
DEBUG=true
```

```sh
envdrift .env.example .env.staging
```

```text
WARNING: extra DEBUG in .env.staging — not declared in .env.example
ERROR: missing STRIPE_WEBHOOK_SECRET in .env.staging — required by .env.example

Summary: 1 error(s), 1 warning(s)
```

不传文件路径时，`envdrift` 默认检查 `.env.example` 与 `.env`。

## 检查项

| 发现项 | 默认级别 | 含义 |
| --- | --- | --- |
| `missing` | error | 契约中的变量未出现在目标文件。 |
| `extra` | warning | 目标文件中存在未在契约中声明的变量。 |
| `value` | warning | 变量值不同；使用 `--values` 启用。 |
| duplicate key | 输入错误 | 同一个 dotenv 文件重复定义变量。 |

工具**绝不会输出变量值**。启用 `--values` 时，只会显示短 SHA-256 指纹，便于比较又不会把凭据写进 CI 日志。

## 常用命令

```sh
# 用一个契约检查多个目标文件。
envdrift .env.example .env.local .env.staging

# 同时检查值是否不同（输出中只含安全指纹）。
envdrift --values .env.example .env.ci

# 将未声明的变量也视为构建失败。
envdrift --fail-on warning .env.example .env.production

# 忽略由仓库外部管理的变量。
envdrift --ignore SENTRY_DSN --ignore CI_JOB_TOKEN .env.example .env.ci

# 输出 JSON，交给后续 CI 步骤处理。
envdrift --format json .env.example .env.staging
```

## GitHub Actions

将以下任务加入工作流：

```yaml
env-contract:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: stable
    - run: go run ./cmd/envdrift .env.example .env.ci.example
```

不要为了通过检查而提交真实密钥。应检查可安全提交的配置契约文件，或者在 CI 中从密钥管理服务临时生成目标文件。

## 支持的 dotenv 语法

- 空行与整行注释
- `KEY=value` 赋值
- `export KEY=value` 赋值
- 单行单引号、双引号值
- 值后以空格分隔的行内注释，例如 `KEY=value # note`

工具刻意拒绝多行 Shell 表达式：比起猜测 Shell 语义，在 CI 中提供确定且容易诊断的检查更重要。

## 本地开发

```sh
go test ./...
go vet ./...
go build ./cmd/envdrift
```

项目只使用 Go 标准库。欢迎提交 Issue 与小而聚焦的 Pull Request，详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 路线图

- [ ] 提供带 PR 注释的 GitHub Action 封装
- [ ] 支持可选的 `.envdrift.yml` 策略文件
- [ ] 提供 Shell 补全脚本

## 许可

MIT，详见 [LICENSE](LICENSE)。

