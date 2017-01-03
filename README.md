# 概述

​	k8s2lb监听kubernetes的service及对应的endpoints，并将其对应映射为neutron lbaasv2中的loadbalancer，实现负载均衡功能。在neutron基础上，与skynet结合，还可以提供浮动IP、VPN、安全组等网络特性。

# 编译

​	基于go语言实现，可以直接使用`go build`进行编译。如下所示：

```shell
mkdir src &&git clone https://github.com/swordboy/k8s2lb.git
export GOPATH=$(pwd)
cd src/k8s2lb&&go build k8s2lb.go
```

​	生成的二进制文件即可用于运行

# 运行配置

通过配置conf.json来指定k8s2lb的运行配置，示例配置如下所示：

```json
{
  "kubernetes" : {
    "k8s_api_root" : "http://192.168.7.203:8080",
    "k8s_cluster_name" : "k8s",
    "kube_config_path": ""
  },
  "neutron" : {
    "neutron_url" : "http://192.168.7.211:9696",
    "default_subnet_id" : "76aa33bc-c9c1-4834-bcfc-aefd28206997"
  },
  "lb_refresh_interval" : 5,
  "use_service_subnet" : false,
  "clean_orphan_loadbalancers": false,
  "auto_clean_orphan_ports" : false,
  "orphan_ports_resync_interval": 1800
}
```



> k8s_api_root: kubernetes master的访问点，例如http://192.168.7.208:8080。目前仅支持http。
>
> k8s_cluster_name：kubernetes集群名称，默认值为k8s，在两个不同kubernetes集群接入同一个neutron网络节点时必须配置。
>
> kube_config_path：连接kubernetes master使用的kubeconfig配置文件路径，与k8s_api_root冲突，两者不能同时配置。
>
> neutron_url：neutron-server的访问点，例如http://192.168.7.210:9696，目前仅支持HTTP，且neutron-server的认证模式需要改为none。
>
> default_subnet_id：服务没有设置子网时，默认为服务设置的子网，必须正确设置。
>
> lb_refresh_interval：由于lb刷新状态每个操作之间必须等到loadbalancer状态稳定为ACTIVE才可操作，此配置指定刷新检查loadbalancer状态的时间间隔。单位为秒。
>
> user_service_subnet：TBD
>
> clean_orphan_loadbalancers：完整同步时，是否自动清理与kubernetes service不对应的所有loadbalancer也进行清理。默认为false。
>
> auto_clean_orphan_ports：是否自动清理neutron中与，kubernetes pod无法对应的port（不包含dhcp等neutron专用port）。默认为false。
>
> orphan_ports_resync_interval：清理orphan ports是，重新全部同步数据的时间间隔。单位为秒。

# TBD

