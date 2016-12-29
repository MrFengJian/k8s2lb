package transformer

import (
        "util"
        "neutron"
        "k8s"
        "fmt"
        "os"
        "time"
        "errors"
        "strings"
        "strconv"
        "k8s.io/client-go/pkg/watch"
        "k8s.io/client-go/pkg/api/v1"
)

const RESTART_INTERVAL = time.Duration(10) * time.Second
const SUBNET_KEY = "skynet/subnet_id"

var CACHE map[string]*v1.Service

func init() {
        CACHE = make(map[string]*v1.Service)
}

func SyncServices(conf *util.Conf) {
        k8sApi := k8s.NewK8sApi(conf)
        neutronApi := neutron.NewNeutronApi(conf.Neutron.NeutronUrl)
        endpointsList, err := k8sApi.ListEndpoints()
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to list endpoints ,will try next turn", err)
                time.Sleep(RESTART_INTERVAL)
                SyncServices(conf)
        }
        serviceList, err := k8sApi.ListServices()
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to list services,will try next turn", err)
                time.Sleep(RESTART_INTERVAL)
                SyncServices(conf)
        }
        lbs, err := neutronApi.ListLoadBalancers()
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to list loadbalancers,will try next turn", err)
                time.Sleep(RESTART_INTERVAL)
                SyncServices(conf)
        }
        lbsMap := make(map[string]*neutron.LoadBalancer)
        for i, s := 0, len(lbs); i < s; i++ {
                lb := lbs[i]
                lbsMap[lb.Name] = &lb
        }
        for i, s := 0, len(serviceList.Items); i < s; i++ {
                service := serviceList.Items[i]
                fmt.Println("service :", service)
                serviceLbName := getLbName(service.ObjectMeta.Name, service.ObjectMeta.Namespace, conf)
                CACHE[serviceLbName] = &service
                delete(lbsMap, serviceLbName)
        }
        for lbName, lb := range lbsMap {
                if conf.CleanOrphanLoadbalancers || strings.HasPrefix(lbName, conf.Kubernetes.K8sClusterName) {
                        fmt.Fprintln(os.Stdout, " loadbalancer", lbName, " is orphan,deleted it automatically")
                        neutronApi.DeleteLoadBalancer(lb.Id)
                }
        }
        for i, s := 0, len(endpointsList.Items); i < s; i++ {
                ep := endpointsList.Items[i]
                fmt.Println("endpoints :", ep)
                if err := syncEp(k8sApi, neutronApi, &ep, watch.Added, conf); err != nil {
                        fmt.Fprintln(os.Stderr, "failed to sync ep ", ep.ObjectMeta.Name, "_", ep.ObjectMeta.Namespace, "just skip", err)
                }
        }
        var endpointsChan <-chan watch.Event
        endpointsChan, err = k8sApi.WatchEndpoints()
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to watch endpoints on ", conf.Kubernetes.K8sApiRoot)
                time.Sleep(RESTART_INTERVAL)
                SyncServices(conf)
        }
        for event := range endpointsChan {
                if ep, ok := event.Object.(*v1.Endpoints); ok {
                        fmt.Println("receive watch endpoints event ", event.Type, "name: ", ep.ObjectMeta.Name, "namespace: ", ep.ObjectMeta.Namespace)
                        syncEp(k8sApi, neutronApi, ep, event.Type, conf)
                }
        }
        fmt.Println("service sync exit abnormally,try restart")
        time.Sleep(RESTART_INTERVAL)
        SyncServices(conf)
}

func syncEp(k8sApi *k8s.K8sApi, neutronApi *neutron.NeutronApi, ep *v1.Endpoints, action watch.EventType, conf *util.Conf) (err error) {
        if (action == watch.Deleted) {
                deleteLoadBalancer(ep, neutronApi, conf)
                return
        }
        lbName := getLbName(ep.ObjectMeta.Name, ep.ObjectMeta.Namespace, conf)
        lbs, err := neutronApi.GetLoadBalanacers(lbName)
        if err != nil {
                return err
        }
        var epLb *neutron.LoadBalancer
        if len(lbs) >= 1 {
                epLb = &lbs[0]
                for _, duplicateLb := range lbs[1:] {
                        neutronApi.DeleteLoadBalancer(duplicateLb.Id)
                }
        }
        var service *v1.Service
        if service, ok := CACHE[lbName]; !ok {
                service, err = k8sApi.GetService(ep.ObjectMeta.Name, ep.ObjectMeta.Namespace)
                if err != nil {
                        return err
                }
                CACHE[lbName] = service
        }
        service = CACHE[lbName]
        servicePorts := service.Spec.Ports

        portsMap := make(map[int32]*v1.ServicePort)
        var cleanLb bool
        for i, s := 0, len(servicePorts); i < s; i++ {
                servicePort := servicePorts[i]
                port := servicePort.TargetPort
                if port.IntVal == 0 {
                        fmt.Fprintln(os.Stdout, lbName, "'s corresponding service's target port is nil,invalid value,skip it ")
                        cleanLb = true
                        continue
                }
                if port.IntVal == 1 {
                        fmt.Fprintln(os.Stdout, lbName, "'s corresponding service's target port is 1,invalid value,skip it ")
                        cleanLb = true
                        continue
                }
                portsMap[port.IntVal] = &servicePort
        }
        if cleanLb {
                neutronApi.DeleteLoadBalancerByName(lbName)
        }
        serviceSubnetId := conf.Neutron.DefaultSubnetId
        if service.ObjectMeta.Annotations[SUBNET_KEY] != "" {
                serviceSubnetId = service.ObjectMeta.Annotations[SUBNET_KEY]
        }
        if epLb != nil {
                serviceSubnetId = epLb.VipSubnetId
        }
        subnet, err := neutronApi.GetSubnet(serviceSubnetId)
        if err != nil {
                return err
        }
        tenantId := subnet.TenantId
        loopInterval := conf.LbRefreshInterval * time.Second
        if epLb == nil {
                if conf.UseServiceSubnet {
                        epLb, err = createServiceSubnetLoadBalancer(neutronApi, conf.Neutron.DefaultSubnetId, tenantId, lbName, service.Spec.ClusterIP)
                }else {
                        epLb, err = createNormalLoadBalancer(neutronApi, serviceSubnetId, tenantId, lbName)
                }
                if err != nil {
                        return errors.New("failed to create service subnet loadbalancer for " + lbName + " with subnet " + conf.Neutron.DefaultSubnetId)
                }
                epLb, err = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                if err != nil {
                        return err
                }
        }
        lbStatus, err := neutronApi.GetLoadBalancerStatus(epLb.Id)
        if err != nil {
                return errors.New("failed to create service subnet loadbalancer for " + lbName + " with subnet " + conf.Neutron.DefaultSubnetId)
        }

        listenersMap := make(map[int32]*neutron.Listener)
        for i, s := 0, len(lbStatus.Listeners); i < s; i++ {
                listener := lbStatus.Listeners[i]
                parts := strings.Split(listener.Name, "_")
                tempPort, err := strconv.Atoi(parts[len(parts) - 1])
                if err != nil {
                        fmt.Fprintln(os.Stderr, listener.Name + " is not a valid kubernetes listener,skip it")
                        continue
                }
                listenersMap[int32(tempPort)] = &listener
        }
        endPointAddresses := make([]v1.EndpointAddress, 0)
        if len(ep.Subsets) > 0 {
                endPointAddresses = ep.Subsets[0].Addresses
        }
        for protocolPort, servicePort := range portsMap {
                expectPoolName := fmt.Sprintf("%s_%d", lbName, protocolPort)
                if listener, ok := listenersMap[protocolPort]; ok {
                        delete(listenersMap, protocolPort)
                        lbPools := lbStatus.Pools
                        existedPools := make(map[string]*neutron.Pool)
                        for i, s := 0, len(lbPools); i < s; i++ {
                                pool := lbPools[i]
                                existedPools[pool.Name] = &pool
                        }
                        if pool, ok := existedPools[expectPoolName]; !ok {
                                pool = &neutron.Pool{}
                                pool.Name = expectPoolName
                                lbAlgorithm := neutron.LB_ROUND_ROBIN_ALGORITHM
                                if service.Spec.SessionAffinity == "ClientIP" {
                                        lbAlgorithm = neutron.LB_SOURCE_IP_ALGORITHM
                                        pool.SessionPersistence = neutron.SesssionPersistence{Type:neutron.LB_SOURCE_IP_ALGORITHM}
                                }
                                pool.LbAlgorithm = lbAlgorithm
                                pool.TenantId = tenantId
                                pool.ListenerId = listener.Id
                                pool.LoadBalancerId = epLb.Id
                                pool.Protocol = string(servicePort.Protocol)
                                fmt.Fprintln(os.Stdout, "try to create pool ", expectPoolName, " with algorithmn ", lbAlgorithm)
                                pool, err = neutronApi.CreatePool(pool)
                                if err != nil {
                                        fmt.Fprintln(os.Stderr, "failed to create pool", expectPoolName)
                                        return err
                                }
                                epLb, err = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                                if err != nil {
                                        fmt.Fprintln(os.Stderr, "failed to create pool", expectPoolName)
                                        return err
                                }
                                for _, endpointAddress := range endPointAddresses {
                                        member := neutron.Member{}
                                        member.TenantId = tenantId
                                        //member.PoolId = pool.Id
                                        member.ProtocolPort = protocolPort
                                        member.SubnetId = serviceSubnetId
                                        member.Weight = 1
                                        member.Address = endpointAddress.IP
                                        fmt.Fprintln(os.Stdout, "create pool ", expectPoolName, "'s member ", endpointAddress.IP, " for backend port ", protocolPort)
                                        neutronApi.CreateMember(pool.Id, &member)
                                        epLb, _ = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                                }
                        }else {
                                existedMembers := make(map[string]*neutron.Member)
                                for i, s := 0, len(pool.Members); i < s; i++ {
                                        member := pool.Members[i]
                                        existedMembers[member.Address] = &member
                                }
                                for _, address := range endPointAddresses {
                                        if _, ok := existedMembers[address.IP]; ok {
                                                delete(existedMembers, address.IP)
                                        }else {
                                                member := neutron.Member{}
                                                member.TenantId = tenantId
                                                member.ProtocolPort = protocolPort
                                                member.SubnetId = serviceSubnetId
                                                member.Weight = 1
                                                member.Address = address.IP
                                                fmt.Fprintln(os.Stdout, "create pool", expectPoolName, "'s member ", address.IP, " for backend port ", protocolPort)
                                                neutronApi.CreateMember(pool.Id, &member)
                                                epLb, _ = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                                        }
                                }
                                for _, leftMember := range existedMembers {
                                        fmt.Fprintln(os.Stdout, "member ", leftMember.Address, "in pool " + expectPoolName, " do not existed,delete it")
                                        neutronApi.DeleteMember(pool.Id, leftMember.Id)
                                        epLb, _ = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                                }
                        }
                }else {
                        listenerReq := neutron.Listener{}
                        listenerReq.Name = expectPoolName
                        listenerReq.ProtocolPort = protocolPort
                        listenerReq.Protocol = string(servicePort.Protocol)
                        listenerReq.TenantId = tenantId
                        listenerReq.LoadBalancerId = epLb.Id
                        fmt.Fprintln(os.Stdout, "try to create listener " + listenerReq.Name)
                        listener, err = neutronApi.CreateListener(&listenerReq)
                        if err != nil {
                                return err
                        }
                        epLb, _ = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                        pool := &neutron.Pool{}
                        pool.Name = expectPoolName
                        lbAlgorithm := neutron.LB_ROUND_ROBIN_ALGORITHM
                        if service.Spec.SessionAffinity == "ClientIP" {
                                lbAlgorithm = neutron.LB_SOURCE_IP_ALGORITHM
                                pool.SessionPersistence = neutron.SesssionPersistence{Type:neutron.LB_SOURCE_IP_ALGORITHM}
                        }
                        pool.LbAlgorithm = lbAlgorithm
                        pool.TenantId = tenantId
                        pool.ListenerId = listener.Id
                        pool.LoadBalancerId = epLb.Id
                        pool.Protocol = string(servicePort.Protocol)
                        fmt.Fprintln(os.Stdout, "try to create pool " + expectPoolName + " with algorithmn " + lbAlgorithm)
                        pool, err = neutronApi.CreatePool(pool)
                        if err != nil {
                                fmt.Fprintln(os.Stderr, "failed to create pool", expectPoolName)
                                return err
                        }
                        epLb, err = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                        if err != nil {
                                fmt.Fprintln(os.Stderr, "failed to create pool", expectPoolName)
                                return err
                        }
                        for _, endpointAddress := range endPointAddresses {
                                member := neutron.Member{}
                                member.TenantId = tenantId
                                member.ProtocolPort = protocolPort
                                member.SubnetId = serviceSubnetId
                                member.Weight = 1
                                member.Address = endpointAddress.IP
                                fmt.Fprintln(os.Stdout, "create pool", expectPoolName, "'s member ", endpointAddress.IP, " for backend port ", protocolPort)
                                neutronApi.CreateMember(pool.Id, &member)
                                epLb, _ = waitLoadBalancer(neutronApi, epLb.Id, loopInterval)
                        }
                }
        }
        for _, leftListener := range listenersMap {
                fmt.Fprintln(os.Stdout, " listener is left and will be deleted:" + leftListener.Name)
                neutronApi.DeleteListenerWithPools(leftListener)
        }
        return nil
}

func waitLoadBalancer(neutronApi *neutron.NeutronApi, lbId string, duration time.Duration) (epLb *neutron.LoadBalancer, err error) {
        epLb, err = neutronApi.GetLoadBalancer(lbId)
        if err != nil {
                return nil, err
        }
        for ; strings.HasPrefix(epLb.ProvisionStatus, "PENDING"); {
                time.Sleep(duration)
                epLb, err = neutronApi.GetLoadBalancer(lbId)
                fmt.Fprintln(os.Stdout, epLb.Name + " 's provision status is " + epLb.ProvisionStatus)
                if err != nil {
                        return nil, err
                }
        }
        return epLb, nil
}

func createServiceSubnetLoadBalancer(neutronApi *neutron.NeutronApi, subnetId, tenantId, lbName, clusterIP string) (*neutron.LoadBalancer, error) {
        loadbalancer := &neutron.LoadBalancer{}
        loadbalancer.Name = lbName
        loadbalancer.TenantId = tenantId
        loadbalancer.Description = "load balancer for " + lbName
        loadbalancer.VipSubnetId = subnetId
        loadbalancer.VipAddress = clusterIP
        return neutronApi.CreateLoadBalancer(loadbalancer)
}

func createNormalLoadBalancer(neutronApi *neutron.NeutronApi, subnetId, tenantId, lbName string) (*neutron.LoadBalancer, error) {
        loadbalancer := &neutron.LoadBalancer{}
        loadbalancer.OpenStackObject = neutron.OpenStackObject{}
        loadbalancer.Name = lbName
        loadbalancer.TenantId = tenantId
        loadbalancer.Description = "load balancer for " + lbName
        loadbalancer.VipSubnetId = subnetId
        return neutronApi.CreateLoadBalancer(loadbalancer)
}

func deleteLoadBalancer(ep *v1.Endpoints, neutronApi *neutron.NeutronApi, conf *util.Conf) {
        delete(CACHE, getLbName(ep.ObjectMeta.Name, ep.ObjectMeta.Namespace, conf))
        neutronApi.DeleteLoadBalancerByName(getLbName(ep.ObjectMeta.Name, ep.ObjectMeta.Namespace, conf))
}

func getLbName(name, namespace string, conf *util.Conf) string {
        return fmt.Sprintf("%s.%s.svc.cluster.local_%s_%s", name, namespace, "kubernetes", conf.Kubernetes.K8sClusterName)
}

