package k8s

import (
        "util"
        "k8s.io/client-go/kubernetes"
        "k8s.io/client-go/tools/clientcmd"
        meta_v1 "k8s.io/client-go/pkg/apis/meta/v1"
        "k8s.io/client-go/pkg/watch"
        "k8s.io/client-go/pkg/api"
        "k8s.io/client-go/pkg/api/v1"
)

type K8sApi struct {
        Client *kubernetes.Clientset
        conf   *util.Conf
}

func NewK8sApi(conf *util.Conf) (*K8sApi) {
        config, _ := clientcmd.BuildConfigFromFlags(conf.Kubernetes.K8sApiRoot, "")
        client, _ := kubernetes.NewForConfig(config)
        return &K8sApi{Client:client, conf:conf}
}

func (self *K8sApi) ListEndpoints() (*v1.EndpointsList, error) {
        return self.Client.Endpoints(api.NamespaceAll).List(v1.ListOptions{})
}

func (self *K8sApi) ListServices() (*v1.ServiceList, error) {
        return self.Client.Core().Services("").List(v1.ListOptions{})
}

func (self *K8sApi) GetService(name, namespace string) (*v1.Service, error) {
        return self.Client.Core().Services(namespace).Get(name, meta_v1.GetOptions{})
}

func (self *K8sApi) WatchEndpoints() (<-chan watch.Event, error) {
        watching, err := self.Client.Endpoints(api.NamespaceAll).Watch(v1.ListOptions{})
        if err != nil {
                return nil, err
        }
        return watching.ResultChan(), nil
}

func (self *K8sApi) ListPods() (*v1.PodList, error) {
        return self.Client.Core().Pods(api.NamespaceAll).List(v1.ListOptions{})
}