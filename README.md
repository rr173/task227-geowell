# 地热井温压剖面异常分层服务 (task227-geowell)

面向地热工程师的纯后端服务：从多次测井的温度、压力与井深数据识别异常井段，
比较不同日期的梯度边界移动，并发布不可变的剖面对比快照。

## 构建与运行

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./cmd/geowell
./geowell --addr :8080 --db geowell.db
```

## 自检（持久化与重启恢复）

```bash
./geowell --smoke-test
```

该命令真实创建井次、导入测点、校正深度、分层、跨次对比并发布快照，
随后关闭并重新打开数据库验证持久化与重启恢复，以 0 退出码结束。

## 标准门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
./geowell --smoke-test
```

## 业务包

- `ingest`   测井模块：接收数据、深度单调校验、幂等指纹。
- `datum`    基准模块：深度基准校正、井口扰动检测。
- `layer`    分层模块：均匀重采样、温压梯度、边界突变检测、异常分类。
- `compare`  对比模块：跨次边界对齐、移动量计算、异常标记。
- `snapshot` 快照模块：版本化对比快照发布与状态机。

## API（前缀 /api）

| 能力 | 入口 | 生产实现 |
| --- | --- | --- |
| 健康检查 | GET /api/health | httpapi |
| 井次列表/创建 | GET/POST /api/wells | httpapi → service.CreateWellRun → store |
| 井次详情/封存 | GET/DELETE /api/wells/{id} | httpapi → service.ArchiveRun → store |
| 测点列表 | GET /api/wells/{id}/points | httpapi → store.ListPoints |
| 深度校正 | POST /api/wells/{id}/correct | httpapi → service.CorrectDepth → datum |
| 分层 | POST /api/wells/{id}/layer | httpapi → service.Layer → layer |
| 井段列表 | GET /api/wells/{id}/segments | httpapi → store.ListSegments |
| 井段确认 | POST /api/segments/{id}/confirm | httpapi → service.ConfirmSegment |
| 跨次对比 | POST /api/compare | httpapi → service.CompareRuns → compare |
| 快照列表 | GET /api/snapshots?well= | httpapi → store.ListSnapshotsByWell |
| 快照详情/发布 | GET/POST /api/snapshots/{id} | httpapi → service.PublishComparison |
| 统计 | GET /api/stats | httpapi |
