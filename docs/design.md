# 实现方案

​	参考[skynet](https://github.com/swordboy/skynet/blob/master/docs/design.md)中的整体实现方案，k8s2lb负责将kubernetes中的serivce映射为neutron lbaasv2中的loadbalancer。kubernetes中的对象与neutron lbaasv2中的对象映射关系如下：

+ service:  neutron lbaasv2 loadbalancer
+ service port:  neutron lbaasv2 listener
+ endpoints: neutron lbaasv2 pool
+ pod ip +targetPort：neutron lbaasv2 member

# 限制

## kubernetes service定义targetPort

前从service定义的targetPort的数字值来作为loadbalancer后端的listener的协议端口，如果使用名称，则无法根据名称找到对应的端口号是什么，无法正常映射为loadbalancer。

# loadbalancer创建延迟

由于neutron lbaasv2中的loadbalancer状态更新的限制，向其添加listener、pool、member都需要等待loadbalancer状态允许时，才能执行。因此，一个完全新建的服务，需要等待一段时间才能正常提供服务。

在测试过程中，还发现loadbalancer已经创建，neutron-dhcp-agent的dnsmasq服务配置已经包含了正确的域名，但是通过服务名称解析有时候还是要等上十几秒。但是在并发负载比较低的时候，可以比较快的解析。通过对neutron-dhcp-agent刷新dnsmasq的机制来看，在并发高的时候，不断刷新dnsmasq配置过程中，会有较长时间dnsmasq服务无法响应。

## 不支持headless service

在kubernetes中存在headless service这一特殊service，它不存在对应的clusterIP。kube-proxy也不会处理这类service，同时这些服务的配置也不会作为环境变量被注入到容器中。它在dns中表现为对应多个后端POD地址的一个域名，负载均衡等访问方式，由用户来决定和扩展。

# TBD：备选服务实现方案

参考了[kuryr与kubernetes](https://blueprints.launchpad.net/kuryr/+spec/kuryr-k8s-integration)集成的实现设计方案，将kubernetes中的serivice也映射为neutron中的某个真实port，而不是只存在于iptables上的一个虚拟IP。

kubernetes的service cluster ip range一般为10.254.0.0/16，将其映射到kubernetes的中网络。保持kubernetes中service ip不变。

