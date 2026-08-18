# go-ratelimitx

纯 Go 限流与配额库：令牌桶、滑动窗口、复合键、租户配额账本。无前端。

## 能力

- `Allow` / `AllowN` / `Peek` / `Reset` / `Reserve`
- 配额 `Charge` / `Refund`（收据防双花；默认可测负债模式）
- 策略：`Rate{Rate, Burst}`、`Window{Size, Limit}`、`QuotaPlan{Period, Soft, Hard}`
- 复合键 `tenant|route|ip`，缺失维度按固定顺序回退
- 可注入时钟、分片存储、拒绝/通过指标

## 安装

```text
go get github.com/LYH2263/go-ratelimitx
```

## 最小示例

```go
eng := ratelimitx.New(
    ratelimitx.WithShards(32),
    ratelimitx.WithDefaultPolicy("default"),
)
_ = eng.Register(ratelimitx.Spec{
    Name: "default",
    Rate: &ratelimitx.Rate{PerSecond: 100, Burst: 20},
})
ok, wait := eng.Allow("acme|/checkout|10.0.0.1", 1)
_, _ = ok, wait
```

测试可用假时钟推进时间，验证 burst 用尽与窗口翻滚。
