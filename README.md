# k8s-database-operator
`k8s-database-operator` 是一个用于 Kubernetes 集群中的数据库管理的操作器（Operator）。它提供了自动化的数据库实例创建、更新和删除功能，简化了数据库管理的复杂性，并提高了管理效率。

## 项目描述
`k8s-database-operator` 是一个基于 Kubernetes 的自定义控制器（Operator），用来自动化管理不同类型（包括 MySQL、PostgreSQL、OceanBase-CE）的数据库实例。通过定义和操作 Kubernetes 的自定义资源（CR），它可以让数据库的部署和管理变得更加简便。操作器支持多种数据库类型，能够处理数据库的生命周期管理，包括创建、备份、恢复、升级和删除。通过结合 Kubernetes 的原生功能和自定义的控制器逻辑，提供了高效且一致的数据库部署与管理体验。

> **注意事项：**
> 1. **OceanBase-CE** 的备份等待进一步实现，目前的方法还有问题
> 2. 备份用到了 **PVC**，需事先创建好对应的 PV 存储类，如 NFS、Ceph 等，否则 PVC 无法成功创建，会导致备份失败

## 开始使用

### 版本要求
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### 部署到集群
**构建并推送镜像到指定的 `IMG` 位置：**

```sh
make docker-build docker-push IMG=<some-registry>/k8s-database-operator:tag
```

**注意：** 这个镜像需要发布到指定的 registry 中，并确保在工作环境中可以拉取到该镜像。如果以上命令无法执行，请检查你的 registry 访问权限。

**将 CRDs 安装到集群中：**

```sh
make install
```

**使用指定的 `IMG` 将 Manager 管理器部署到集群中：**

```sh
make deploy IMG=<some-registry>/k8s-database-operator:tag
```

> **注意**: 如果遇到 RBAC 错误，可能需要授予集群管理员权限或以管理员身份登录。

**创建自定义资源的实例**

可以使用 config/samples 中的示例配置（也可以适当进行修改）：

```sh
kubectl apply -k config/samples/
```

>**注意**: 确保示例配置有默认值，以便能进行测试。

### 删除自定义资源
**删除集群中的自定义资源实例 (CRs)：**

```sh
kubectl delete -k config/samples/
```

**从集群中删除 APIs（CRDs）：**

```sh
make uninstall
```

**从集群中删除自定义的控制器实例：**

```sh
make undeploy
```

## 分发项目

以下是构建安装程序并分发此项目给用户的步骤。

1. 为在 registry 中构建和发布的镜像，构建安装程序：

```sh
make build-installer IMG=<some-registry>/k8s-database-operator:tag
```

注意: 上面的 make 目标会在 dist 目录中生成一个 'install.yaml' 文件，这个文件包含了使用 Kustomize 构建的所有资源，可以在没有依赖项的情况下安装本项目。

2. 使用安装程序

用户可以运行以下命令来安装项目，yaml 文件的具体地址需要按实际情况指定一下：

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/k8s-database-operator/<tag or branch>/dist/install.yaml
