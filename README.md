

方案描述:
1. 使用了[第三方库](https://github.com/dgryski/go-maglev)的 maglev 算法
2. 计算 maglev index 时需要一个 hash, 


细节:
- 配置加载
    - 支持的配置
        - header 
        - http cookie
        - ip 
    - from config
    - from Istio
- maglev 初始化
    - 1. 库:
        - 使用[这个库](https://github.com/dgryski/go-maglev)的 maglev 算法
        - 使用[这个分支] 的 `protocolVariable` 接口, 
    - 保存 maglev 信息, 在 `ChooseHost` 接口时根据因子(header/cookie/ip) 返回hash 
        (uint64), 传入maglev 库
    - 2. 初始化:
        - 在[这里](https://github.com/mosn/mosn/blob/feature-istio_adapter/pkg/upstream/cluster/loadbalancer.go#L96) 
            加一个 `maglevLoadBalancer` 结构体, 保存 maglev 库需要的信息
        - new `maglevLoadBalancer` 时, 根据 host 列表计算 maglev 表
- 路由
    - 1. `ChooseHash` 接口执行时, 根据因子(header/cookie/ip)