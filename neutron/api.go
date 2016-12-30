package neutron

import (
        "net/http"
        "io/ioutil"
        "encoding/json"
        "fmt"
        "os"
        "strings"
)

const JSON_TYPE = "application/json"
const LB_PATH = "/v2.0/lbaas/loadbalancers"
const LISTENER_PATH = "/v2.0/lbaas/listeners"
const POOL_PATH = "/v2.0/lbaas/pools"

type NeutronApi struct {
        NeutronUrl string
}

func NewNeutronApi(neutronUrl string) (api *NeutronApi) {
        return &NeutronApi{NeutronUrl:neutronUrl}
}

func (api *NeutronApi) GetSubnet(subnetId string) (subnet *OpenStackSubnet, err error) {
        url := api.NeutronUrl + "/v2.0/subnets/" + subnetId
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response SubnetResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.Subnet, nil
}

//callDelete wraps http delete request
func callDelete(url string) (err error) {
        req, err := http.NewRequest(http.MethodDelete, url, nil)
        if err != nil {
                return nil
        }
        _, err = http.DefaultClient.Do(req)
        if err != nil {
                return err
        }
        return nil
}

func (api *NeutronApi)ListLoadBalancers() (lbs []LoadBalancer, err error) {
        url := fmt.Sprintf("%s%s", api.NeutronUrl, LB_PATH)
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var lbsResponse LoadBalancersResponse
        json.Unmarshal(data, &lbsResponse)
        return lbsResponse.Lbs, nil
}

func (api *NeutronApi) GetLoadBalanacers(name string) (lbs []LoadBalancer, err error) {
        url := fmt.Sprintf("%s%s?name=%s", api.NeutronUrl, LB_PATH, name)
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var lbsResponse LoadBalancersResponse
        json.Unmarshal(data, &lbsResponse)
        return lbsResponse.Lbs, nil
}

func (api *NeutronApi) DeleteMember(poolId, memberId string) {
        url := fmt.Sprintf("%s%s/%s/members/%s", api.NeutronUrl, POOL_PATH, poolId, memberId)
        callDelete(url)
}

func (api *NeutronApi) DeletePool(poolId string) {
        url := fmt.Sprintf("%s%s/%s", api.NeutronUrl, POOL_PATH, poolId)
        callDelete(url)
}

func (api *NeutronApi) DeleteListener(listenerId string) {
        url := fmt.Sprintf("%s%s/%s", api.NeutronUrl, LISTENER_PATH, listenerId)
        callDelete(url)
}

func (api *NeutronApi)DeleteLoadBalancer(id string) error {
        var lb *LoadBalancer
        lb, err := api.GetLoadBalancerStatus(id)
        if err != nil {
                return err
        }
        for _, pool := range lb.Pools {
                poolId := pool.Id
                for _, member := range pool.Members {
                        api.DeleteMember(poolId, member.Id)
                }
                api.DeletePool(poolId)
        }
        for _, listener := range lb.Listeners {
                listenerId := listener.Id
                api.DeleteListener(listenerId)
        }
        url := fmt.Sprintf("%s%s/%s", api.NeutronUrl, LB_PATH, id)
        callDelete(url)
        return nil
}

func (api *NeutronApi) GetLoadBalancerStatus(id string) (lb *LoadBalancer, err error) {
        url := fmt.Sprintf("%s%s/%s/statuses", api.NeutronUrl, LB_PATH, id)
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var statusResp LoadBalancerStatusResponse
        json.Unmarshal(data, &statusResp)
        return &statusResp.Statuses.Lb, nil
}

func (api *NeutronApi) DeleteLoadBalancerByName(name string) error {
        lbs, err := api.GetLoadBalanacers(name)
        if err != nil {
                fmt.Fprintln(os.Stderr, "failed to query lb ", name, "while delete it,ignore error", err)
        }
        for _, lb := range lbs {
                api.DeleteLoadBalancer(lb.Id)
        }
        return nil
}

func (api *NeutronApi) CreateLoadBalancer(lb *LoadBalancer) (loadbalancer *LoadBalancer, err error) {
        requestBody := LoadBalancerResponse{LoadBalanacerBody:*lb}
        data, err := json.Marshal(requestBody)
        if err != nil {
                return nil, err
        }
        url := api.NeutronUrl + LB_PATH
        resp, err := http.Post(url, JSON_TYPE, strings.NewReader(string(data)))
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response LoadBalancerResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.LoadBalanacerBody, nil
}

func (api *NeutronApi) GetLoadBalancer(lbId string) (*LoadBalancer, error) {
        url := fmt.Sprintf("%s%s/%s", api.NeutronUrl, LB_PATH, lbId)
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response LoadBalancerResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.LoadBalanacerBody, nil
}

func (api *NeutronApi) CreatePool(pool *Pool) (*Pool, error) {
        requestBody := PoolResponse{PoolBody:*pool}
        data, err := json.Marshal(requestBody)
        url := api.NeutronUrl + POOL_PATH
        resp, err := http.Post(url, JSON_TYPE, strings.NewReader(string(data)))
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response PoolResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.PoolBody, nil
}

func (api *NeutronApi) CreateMember(poolId string, member *Member) (*Member, error) {
        url := fmt.Sprintf("%s%s/%s/members", api.NeutronUrl, POOL_PATH, poolId)
        requestBody := MemberResponse{MemberBody:*member}
        data, err := json.Marshal(requestBody)
        resp, err := http.Post(url, JSON_TYPE, strings.NewReader(string(data)))
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response MemberResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.MemberBody, nil
}

func (api *NeutronApi) CreateListener(listener *Listener) (*Listener, error) {
        requestBody := ListenerResponse{ListenerBody:*listener}
        data, err := json.Marshal(requestBody)
        url := api.NeutronUrl + LISTENER_PATH
        resp, err := http.Post(url, JSON_TYPE, strings.NewReader(string(data)))
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response ListenerResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return &response.ListenerBody, nil
}

func (api *NeutronApi) DeleteListenerWithPools(listener *Listener) error {
        listenerId := listener.Id
        for _, pool := range listener.Pools {
                poolId := pool.Id
                for _, member := range pool.Members {
                        api.DeleteMember(poolId, member.Id)
                }
                api.DeletePool(poolId)
        }
        api.DeleteListener(listenerId)
        return nil
}

func (api *NeutronApi) GetPorts() (ports []OpenStackPort, err error) {
        url := api.NeutronUrl + "/v2.0/ports"
        resp, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        var response PortsResponse
        err = json.Unmarshal(data, &response)
        if err != nil {
                return nil, err
        }
        return response.Ports, nil
}

func (api *NeutronApi) DeletePort(portId string) (err error) {
        url := api.NeutronUrl + "/v2.0/ports/" + portId
        err = callDelete(url)
        if err != nil {
                return err
        }
        return nil
}