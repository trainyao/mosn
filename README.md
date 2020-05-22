

方案描述:
1. 使用了[第三方库](https://github.com/dgryski/go-maglev)的 maglev 算法
    - 加载的时候传 host 名数组到库的 `New` 方法进行初始化
    - 路由时, 将一个 连接/请求的 hash 值(uint64) 传进去 `Lookup` 方法, 会返回一个 index, 该 index 就是被选中的 host 数组的索引
    - 服务 down 掉, 调用库的 `Rebuild` 方法传入 down 掉的 host 的索引, 后面该 host 的连接/请求再次请求 `Lookup` 方法会被路由到其他 host 原有的连接/请求不会被影响
    - host 增加时, 需要调用库的 `New` 方法,重新生成 maglev 信息, 此时*所有的连接/请求都会收到影响* (这里有个疑问, 原有的长连接应该如何处理?)
2. 计算 maglev hash
    - xds 支持配置 header / http cookie / ip (一个或多个) 作为生成 hash 的因子
    - Istio 只允许其中之一作为 hash 生成的因子
    - 综上, MOSN 实现好多个因子的 hash 生成, 也能覆盖到 Istio 会传一个因子过来的场景
    - MOSN 里还会将 hash 因子的获取做的比较通用, 适合各种的协议, 参考 [这个接口](https://github.com/mosn/mosn/pull/1107)


细节:
- 配置加载
    - 支持的配置
        - header 
        - http cookie
        - ip
    - from config
        - 将配置保存到 [这里](https://github.com/mosn/mosn/blob/feature-istio_adapter/pkg/config/v2/route.go#L54), 增加一个 `HashPolicy` 数组 
    - from Istio
        - [转换 xds 配置的地方](https://github.com/mosn/mosn/blob/feature-istio_adapter/pkg/xds/conv/convertxds.go#L840) 保存 `HashPolicy` 信息
- maglev 初始化
    - 在[这里](https://github.com/mosn/mosn/blob/feature-istio_adapter/pkg/upstream/cluster/loadbalancer.go#L96) 加一个 `maglevLoadBalancer` 结构体, 保存 maglev 库需要的信息
    - new `maglevLoadBalancer` 时, 传入 host 名数组, 计算 maglev 表
- 路由
    1. `ChooseHash` 接口执行时, 根据因子(header/cookie/ip), 调用 [这个接口](https://github.com/mosn/mosn/pull/1107) 里的 `GetProtocolResource` 方法, 获取对应的值
    2. 将值拼成一个字符串, 竖线分割: [header value]|[http cookie value]|[remote ip], 调用 `github.com/dchest/siphash` 库将字符串生成 hash 值
        - 在考虑只有ip的情况下, 是不是可以直接将 ip 转换为 uint64 作为 hash 值
    3. hash 值传入 `Lookup` 方法生成 index, 返回 host index
    4. 如果有 host down 掉了, 应该调用 `Rebuild` 方法将其下线
        - 这里目前还没想到怎么改
    5. 根据 Istio http cookie 功能, 如果没有该 cookie, 会生成一个
        - 这个目前还没想到怎么改
- 

todo
[x]maglev库性能分析
[x]host down 选择segmenttree 返回临近节点
[x]不拼接, 直接使用单个hash因子
[ ]mosn 原生配置调通
[ ]http2 cookie 逻辑
[ ]mgv info not match, prevent returning nil host pkg/upstream/cluster/loadbalancer.go:215 when choosing host
[ ]单元测试
