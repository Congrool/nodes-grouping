# NodesGrouping

## 采用operator-sdk构建
相关命令
```bash
# 初始化项目
$ operator-sdk init --domain=harmonycloud.io --repo=github.com/Congrool/nodes-grouping

# 创建API
$ operator-sdk create api --group=cluster --version=v1alpha1 --kind=Cluster
$ operator-sdk create api --group=policy --version=v1alpha1 --king=PropagationPolicy

# 修改crd type后，重新生成deepcopy函数
$ make generate

# 在k8s中注册crd
$ make install
```