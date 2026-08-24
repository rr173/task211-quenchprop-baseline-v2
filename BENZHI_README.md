基于 Go 实现的低温超导磁体淬灭传播分析服务，一款纯后端分析服务，处理多通道放电波形、延迟校准与淬灭传播诊断。

# BENZHI 评测说明 · task211-quenchprop

低温超导磁体淬灭传播分析服务（纯 Go 后端，SQLite 持久化，无前端）。

## 运行命令

```bash
export GOTOOLCHAIN=local
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet   ./...
CGO_ENABLED=0 go test  ./...
go run ./cmd/quenchprop --smoke-test
```

## HTTP API

监听 `:8080`，全部路由带 `/api` 前缀（20 个端点）：

- `POST /api/topologies` / `GET /api/topologies/{id}`
- `POST /api/experiments` / `GET /api/experiments` / `GET /api/experiments/{id}`
- `POST /api/experiments/{id}/transition`（状态机流转）
- `POST /api/experiments/{id}/channels` / `GET /api/experiments/{id}/channels`
- `POST /api/experiments/{id}/topology`（绑定拓扑）
- `POST /api/experiments/{id}/waveforms`（幂等摄入）/ `GET /api/experiments/{id}/waveforms`
- `POST /api/experiments/{id}/calibrate`（延迟校准）
- `POST /api/experiments/{id}/analyze`（分析流水线）
- `GET /api/experiments/{id}/analyses` / `GET /api/experiments/{id}/anomalies`
- `POST /api/analyses/{id}/transition`（发布/替代）/ `GET /api/analyses/{id}`
- `POST /api/anomalies/{id}/transition` / `GET /api/anomalies/{id}`
- `GET /api/health`

## Docker 双架构

```bash
bash build_benzhi_docker.sh task211-quenchprop linux/amd64
bash build_benzhi_docker.sh task211-quenchprop linux/arm64
docker run --rm --platform linux/amd64 task211-quenchprop:latest --smoke-test
docker run --rm --platform linux/arm64 task211-quenchprop:latest --smoke-test
```

## --smoke-test 契约

`go run ./cmd/quenchprop --smoke-test`（及容器内 `/app/quenchprop --smoke-test`）不启动
长驻服务，而是用临时数据库跑完整端到端场景并以 0 退出码结束：

- 建 4 分段线性拓扑 + 5 通道（含 1 坏道）；
- 摄入 5 条含同步校准脉冲的淬灭波形（含 1 条重复验证幂等）；
- 校准估计各通道延迟（容差 200µs 内）；
- 分析：起点=seg-1、传播速度 5.000 m/s、受影响分段=4、保护窗口超限（breach）；
- 发布分析包（草稿→复核→发布）；
- 封存试验后写入被拒（`ErrSealed`）。

任一断言失败返回退出码 1。
