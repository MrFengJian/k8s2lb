package transformer

import (
        "k8s2lb/util"
        "fmt"
        "os"
        "k8s2lb/k8s"
        "k8s2lb/neutron"
        "time"
        "strings"
)

func SyncPods(conf *util.Conf) error {
        defer func() {
                if err := recover(); err != nil {
                        fmt.Fprintln(os.Stderr, "failed to sync pods's ports ", err)
                        SyncPods(conf)
                }
        }()
        k8sApi := k8s.NewK8sApi(conf)
        neutronApi := neutron.NewNeutronApi(conf.Neutron.NeutronUrl)
        podList, err := k8sApi.ListPods()
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to list pods while sync pods,try restart")
                time.Sleep(RESTART_INTERVAL)
                SyncPods(conf)
        }
        ports, err := neutronApi.GetPorts()
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to list ports while sync pods,try restart")
                time.Sleep(RESTART_INTERVAL)
                SyncPods(conf)
        }
        existedPods := make(map[string]string)
        for i, s := 0, len(podList.Items); i < s; i++ {
                pod := podList.Items[i]
                existedPods[fmt.Sprintf("%s-%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)] = pod.Status.PodIP
        }
        for i, s := 0, len(ports); i < s; i++ {
                port := ports[i]
                //skip port used by dhcp-agent or lbaas-agent or any other ports
                if strings.HasPrefix(port.DeviceOwner, "neutron") {
                        continue
                }
                //skip port that's not like kubernetes pod's ports
                if len(strings.Split(port.Name, "_")) != 2 {
                        continue
                }
                if len(port.FixedIPS) > 0 {
                        if podIP, ok := existedPods[port.Name]; ok {
                                notDelete := true
                                for _, fixIP := range port.FixedIPS {
                                        if podIP == fixIP.IpAddress {
                                                notDelete = false
                                        }
                                }
                                if notDelete {
                                        neutronApi.DeletePort(port.Id)
                                }
                        }

                }
        }
        return nil
}


