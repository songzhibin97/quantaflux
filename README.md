# quantaflux

```
quantaflux/
├── cmd/
│   └── quantaflux/
│       └── main.go          # 主入口，依赖注入和系统启动
├── configs/
│   └── config.json          # 配置文件
├── internal/
│   ├── ai/                  # AI 分析模块
│   ├── configs/             # 配置加载与校验
│   ├── data/                # 数据收集与存储
│   ├── models/              # 数据模型定义
│   ├── risk/                # 风险管理
│   ├── trading/             # 交易执行
│   └── utils/               # 工具类（HTTP 请求等）
└── scripts/                 # 部署或测试脚本
```