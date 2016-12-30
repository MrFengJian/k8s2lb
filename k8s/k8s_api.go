package k8s

import (
        "k8s2lb/util"
        "k8s.io/client-go/1.5/kubernetes"
        "k8s.io/client-go/1.5/tools/clientcmd"
        "k8s.io/client-go/1.5/pkg/watch"
        "k8s.io/client-go/1.5/pkg/api"
        "k8s.io/client-go/1.5/pkg/api/v1"
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
        return self.Client.Endpoints(api.NamespaceAll).List(api.ListOptions{})
}

func (self *K8sApi) ListServices() (*v1.ServiceList, error) {
        return self.Client.Core().Services("").List(api.ListOptions{})
}

func (self *K8sApi) GetService(name, namespace string) (*v1.Service, error) {
        return self.Client.Core().Services(namespace).Get(name)
}

func (self *K8sApi) WatchEndpoints() (<-chan watch.Event, error) {
        watching, err := self.Client.Endpoints(api.NamespaceAll).Watch(api.ListOptions{})
        if err != nil {
                return nil, err
        }
        return watching.ResultChan(), nil
}

func (self *K8sApi) ListPods() (*v1.PodList, error) {
        return self.Client.Core().Pods(api.NamespaceAll).List(api.ListOptions{})
}