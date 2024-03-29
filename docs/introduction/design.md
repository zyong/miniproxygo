
# 代理服务客户端
## 概念
- 代理服务客户端：指在用户请求产生的环境中有个本地的代理服务程序，负责解决本地的数据加密
- 代理服务服务端：指在外网有个服务端程序，负责真实的请求代理


## 需求
- 实现跨GFW的网络代理服务
  - 支持http和https协议
  - 支持其他应用协议，实现视频和语音的传输
- 实现对客户端配置需求最小化
  - 极大减少对用户不必要的操作干扰
- 实现用户信息安全的最大化
  - 避免用户痕迹的留存
  - 最大的保证用户访问的信息安全


## 设计方案
### PAC文件

* 构建自定义的pac文件

pac文件用来实现浏览器或操作系统的请求转发

### 架构介绍
![Miniproxy架构](../images/miniproxy-arch.jpg)
 
### 功能组件
Miniproxy包括数据平面和控制平面的多个组件
- 数据平面
    - Miniproxy Local 本地客户端
    - Miniproxy Server 转发服务端
- 控制平面
    - [Dashboard] 为用户提供图形化控制界面和监控界面

### 代理服务启动流程
- 设置配置项
- 初始化配置实例
- 解析配置参数
- 实例化服务
- 服务启动
- 实例化加密算法
- 判断启动服务类型
  - 本地服务
  - 远程服务
- 启动服务



### 代理服务客户端执行流程
1. 客户端应用发起网络请求
2. 请求通过pac文件设置判断，需要走代理的走代理服务，不需要走代理的直接发起请求
3. 请求以Socks5协议转发到代理服务客户端
4. 代理服务客户端将请求的域名和数据读取出来，将数据部分加密通过与服务端之间的连接转发出去
5. 代理服务客户端等待代理服务服务端的应答数据
6. 代理服务客户端收到应答数据并解密应答数据
7. 应答数据被转发给客户端


### 代理服务客户端内部逻辑
- 启动客户端逻辑
- 检查客户端监听地址
- 检查客户端加密算法
- 检查客户端密码
- 生成加密对象
- 循环等待请求
- 接收请求
  - 生成包装的连接对象
  - 执行连接处理协程
  - 第一次读取请求数据，分析数据内容，针对第一次请求进行回复
  - 第二次读取请求数据，分析数据内容，针对第二次请求进行回复
  - 读取请求的数据部分和请求头部分，并将请求数据加密转发给服务端
  - 服务端基于请求内容建立到远端的连接，读取远端连接返回数据，将返回数据通过加密后回写给本地代理服务
  - 本地代理服务解密后，将结果返回给连接客户


### 模块化
  - 支持多种协议模块
    - HTTPS
    - Socks5 （优先）
  - 支持多种加密算法
    - AES-256
  - 多种环境配置
    - 客户端
    - 服务端
  - 日志模块
    - 独立日志模块


### 客户端和服务端连接
#### 两种实现
- 连接池    （升级迭代）
- 每次新建连接 （初期）


# 代理服务服务端

## 服务端代理服务内部逻辑 
- 设置配置项
- 初始化配置实例
- 处理配置参数
- 实例化服务
- 启动服务
- 循环等待请求
- 接收请求
  - 创建读取循环协程
  - 创建写入循环协程
- 读取请求的数据部分和请求头部分，解密请求体，通过请求体解析外部请求资源
- 建立到外部的连接，通过连接转发请求体给对方，等待对方应答
- 收到应答数据，将应答数据加密通过channel通知写入客户端请求socket


