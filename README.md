# task211-quenchprop · 低温超导磁体淬灭传播分析服务

面向磁体工程师的淬灭传播分析后端服务：采集多通道放电波形，做延迟校准，
定位淬灭起点，沿线圈拓扑计算传播速度，评估保护窗口并发布分析包。

## 业务域

低温超导磁体在失超（quench）时局部温度/电压急剧上升，并从起点沿线圈拓扑传播。
本服务完成整条分析流水线：

1. 登记放电试验（准备 → 采集 → 分析中 → 已确认 → 封存）；
2. 登记线圈拓扑（分段 + 邻接边）与测量通道（在线/坏道/停用）；
3. 幂等摄入多通道波形（时间轴单调校验 + 采样率一致 + 指纹去重）；
4. 以最陡边对齐做多通道延迟校准；
5. 检测淬灭起点、分类异常段、沿拓扑图遍历计算传播速度；
6. 计算保护窗口并判定是否超出磁体保护能力；
7. 生成分析包（草稿 → 复核 → 发布 → 替代）。

## 标准命令

```bash
export GOTOOLCHAIN=local
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet   ./...
CGO_ENABLED=0 go test  ./...
go run ./cmd/quenchprop --smoke-test   # 端到端自测，0 退出码即通过
go run ./cmd/quenchprop --addr :8080 --db ./quenchprop.db   # 启动 HTTP 服务
```

## API 入口

全部路由带 `/api` 前缀，共 20 个端点：试验 CRUD 与流转、拓扑登记、通道登记、
波形摄入、校准、分析、分析包发布、异常段流转、健康自检。

## 持久化

SQLite（`modernc.org/sqlite` v1.52.0，纯 Go 驱动，CGO 无关，WAL 模式），9 表：
`experiments / topologies / topology_segments / topology_edges / channels /
waveforms / waveform_samples / anomalies / analyses`。波形指纹 UNIQUE 幂等，
乐观锁 `version` 防并发冲突，封存试验拒写，服务以文件 DB 启动时天然具备重启恢复。

## Docker 双架构

```bash
bash build_benzhi_docker.sh task211-quenchprop linux/amd64
bash build_benzhi_docker.sh task211-quenchprop linux/arm64
docker run --rm task211-quenchprop:latest --smoke-test
```
