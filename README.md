# CRD Demo - WaitDeployment Controller

这是一个 Kubernetes 自定义资源定义（CRD）的演示项目，实现了一个名为 `WaitDeployment` 的自定义控制器。

## 项目简介

WaitDeployment 是一个增强型的 Kubernetes Deployment 控制器，它在创建或更新 Deployment 之前会先执行 TCP 探测检查。这样可以确保依赖的服务已经就绪后再启动应用程序，避免因依赖服务未就绪而导致的启动失败。

## 功能特性

- **TCP 探测检查**：在创建 Deployment 之前，先通过 TCP 连接检测依赖服务的可用性
- **自动管理 Deployment**：基于 WaitDeployment CRD 自动创建和更新对应的 Kubernetes Deployment
- **事件记录**：记录资源同步和错误事件到 Kubernetes 事件系统
- **工作队列机制**：使用限速工作队列处理资源变更，支持并发处理

## 项目结构

```
.
├── main.go                    # 程序入口
├── go.mod                     # Go 模块依赖
├── kube/                      # Kubernetes 相关代码
│   ├── apis/                  # CRD API 定义
│   │   └── qbox/v1alpha1/     # WaitDeployment CRD 定义
│   │       └── types.go       # CRD 类型定义
│   ├── client/                # 自动生成的客户端代码
│   ├── config.go              # 配置结构定义
│   ├── controller.go          # 控制器核心逻辑
│   └── signals/               # 信号处理
├── utils/                     # 工具函数
│   ├── defaults.go            # 默认值设置
│   └── yaml.go                # YAML 配置加载
├── hack/                      # 代码生成脚本
│   └── update-codegen.sh      # 客户端代码生成脚本
└── artifacts/                 # 部署相关文件
```

## 核心概念

### WaitDeployment CRD

WaitDeployment 资源定义包含以下字段：

```go
type Waitdeployment struct {
    metav1.TypeMeta
    metav1.ObjectMeta
    Spec      appv1.DeploymentSpec    // 标准的 Deployment Spec
    Status    appv1.DeploymentStatus  // Deployment 状态
    WaitProbe WaitProbe               // 探测配置
}

type WaitProbe struct {
    Address string        // 探测地址，格式：host:port
    Timeout time.Duration // 超时时间
}
```

### 工作流程

1. 用户创建或更新 WaitDeployment 资源
2. 控制器检测到资源变更，将其加入工作队列
3. Worker 从队列中获取任务
4. 执行 TCP 探测检查（根据 WaitProbe 配置）
5. 探测成功后，创建或更新对应的 Deployment
6. 更新 WaitDeployment 状态
7. 记录事件

## 环境要求

- Go 1.13+
- Kubernetes 集群（v1.19+）
- kubectl 命令行工具

## 安装部署

### 1. 编译项目

```bash
go build -o waitdeployment-controller .
```

### 2. 配置文件

创建配置文件 `etc/config.yaml`：

```yaml
kube:
  outCluster: true                          # 是否在集群外运行
  configPath: ~/.kube/config                # kubeconfig 路径
```

如果在集群内运行，设置 `outCluster: false`，控制器将使用 Service Account 认证。

### 3. 部署 CRD

首先需要在 Kubernetes 集群中注册 WaitDeployment CRD（CRD 定义文件需要根据 types.go 生成）。

### 4. 运行控制器

```bash
./waitdeployment-controller -f etc/config.yaml
```

或使用默认配置文件路径：

```bash
./waitdeployment-controller
```

## 使用示例

创建一个 WaitDeployment 资源：

```yaml
apiVersion: qbox.com/v1alpha1
kind: Waitdeployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
waitProbe:
  address: "mysql-service:3306"    # 等待 MySQL 服务就绪
  timeout: 5s                       # 超时时间 5 秒
```

控制器会先尝试连接 `mysql-service:3306`，成功后才会创建 Deployment。

## 开发指南

### 代码生成

当修改 CRD 定义（`kube/apis/qbox/v1alpha1/types.go`）后，需要重新生成客户端代码：

```bash
./hack/update-codegen.sh
```

这会自动生成：
- DeepCopy 方法
- Clientset
- Informers
- Listers

### 控制器架构

控制器采用标准的 Kubernetes Controller 模式：

1. **Informer**：监听 WaitDeployment 和 Deployment 资源变更
2. **Lister**：从本地缓存中查询资源
3. **WorkQueue**：限速工作队列，避免同时处理相同资源
4. **EventHandler**：处理资源的增删改事件
5. **Reconcile Loop**：调谐循环，确保实际状态与期望状态一致

### 主要组件

- **Client**：控制器客户端，管理 Kubernetes 客户端和 Informer
- **syncHandler**：核心同步逻辑，处理单个 WaitDeployment 资源
- **checkAndStartDeployment**：执行 TCP 探测并创建/获取 Deployment
- **doTCPProbe**：TCP 连接探测实现

## 技术栈

- **Kubernetes Client-go**：Kubernetes Go 客户端库
- **k8s.io/apimachinery**：Kubernetes API 机制
- **k8s.io/code-generator**：代码生成工具
- **WorkQueue**：限速工作队列
- **Informer**：资源监听和缓存机制

## 配置参数

控制器支持以下命令行参数：

- `-f`：配置文件路径（默认：`etc/config.yaml`）
- 其他 klog 标准参数（如 `-v` 控制日志级别）

## 监控和日志

控制器使用 `klog` 记录日志，支持以下日志级别：

```bash
# 设置日志级别为 4（详细日志）
./waitdeployment-controller -v=4
```

事件记录到 Kubernetes Events，可以通过以下命令查看：

```bash
kubectl describe waitdeployment <name>
kubectl get events
```

## 故障排查

### 常见问题

1. **探测失败**
   - 检查 WaitProbe 地址是否正确
   - 确认依赖服务是否已启动
   - 调整超时时间

2. **Deployment 未创建**
   - 查看控制器日志
   - 检查 WaitDeployment 资源状态
   - 确认控制器有足够的 RBAC 权限

3. **资源冲突**
   - 确保 Deployment 名称与 WaitDeployment 名称一致
   - 检查是否有同名的独立 Deployment 存在

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目仅用于学习和演示目的。

## 相关资源

- [Kubernetes Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [sample-controller](https://github.com/kubernetes/sample-controller)
- [client-go](https://github.com/kubernetes/client-go)
- [code-generator](https://github.com/kubernetes/code-generator)
