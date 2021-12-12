# NodesGrouping

## 采用operator-sdk构建
创建API
```bash
# 初始化项目
$ operator-sdk init --domain=kubeedge.io --repo=github.com/Congrool/nodes-grouping

# 创建API
$ operator-sdk create api --group=group --version=v1alpha1 --kind=NodeGroup
$ operator-sdk create api --group=policy --version=v1alpha1 --king=PropagationPolicy
```

## 常用命令
```bash
# 生成deepcopy函数
$ make generate

# 生成register函数
$ hack/update_codegen.sh

# 生成crd资源
$ make manifests

# 编译所有组件
$ make all 

# 构建所有组件镜像
$ make images

# 注册CRD对象
$ make install

# 部署node-group-controller-manager
$ make deploy
```