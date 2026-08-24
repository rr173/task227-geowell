基于 Go 实现的地热井温压剖面异常分层 Web 项目，一款后端服务，完成深度基准校正、温压梯度突变分层与异常井段快照发布。

# BENZHI 评测说明 — task227-geowell

## 评测构建命令

```bash
bash build_benzhi_docker.sh docker-baseline-env linux/amd64
bash build_benzhi_docker.sh docker-baseline-env linux/arm64
```

## 双架构 smoke 契约

容器 ENTRYPOINT 固定为 `/out/geowell`，默认 CMD 为 `--smoke-test`。
评测仅传 flag、不追加路径参数：

```bash
docker run --rm --platform linux/amd64  docker-baseline-env:amd64  --smoke-test
docker run --rm --platform linux/arm64  docker-baseline-env:arm64  --smoke-test
```

## API 入口

所有路由以 `/api` 前缀暴露（健康检查 `GET /api/health`、井次 `GET/POST /api/wells`、
分层 `POST /api/wells/{id}/layer`、跨次对比 `POST /api/compare`、快照发布 `POST /api/snapshots/{id}` 等）。

## 持久化

SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 无关），建表：well_runs / measure_points /
segments / comparison_snapshots。重启后恢复未完成分层，相同井次/测点序号幂等。
