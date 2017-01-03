package util

import (
        "os"
        "io/ioutil"
        "encoding/json"
        "time"
)

type KubernetesConfig struct {
        K8sApiRoot     string `json:"k8s_api_root"`
        K8sClusterName string `json:"k8s_cluster_name"`
        KubeConfig     string `json:"kube_config_path"`
}

type NeutronConfig struct {
        NeutronUrl      string `json:"neutron_url"`
        DefaultSubnetId string `json:"default_subnet_id"`
}

type Conf struct {
        Neutron                   NeutronConfig `json:"neutron"`
        Kubernetes                KubernetesConfig `json:"kubernetes"`
        LbRefreshInterval         time.Duration `json:"lb_refresh_interval"`
        UseServiceSubnet          bool `json:"use_service_subnet"`
        AutoCleanOrphanPorts      bool `json:"auto_clean_orphan_ports"`
        OrphanPortsResyncInterval time.Duration `json:"orphan_ports_resync_interval"`
        CleanOrphanLoadbalancers  bool `json:"clean_orphan_loadbalancers"`
}

func LoadConf(confPath string) (conf *Conf, err error) {
        file, err := os.Open(confPath)
        if err != nil {
                return nil, err
        }
        data, err := ioutil.ReadAll(file)
        if err != nil {
                return nil, err
        }
        json.Unmarshal(data, &conf)
        return conf, nil
}