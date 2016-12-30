package neutron

const LB_ROUND_ROBIN_ALGORITHM = "ROUND_ROBIN"
const LB_SOURCE_IP_ALGORITHM = "SOURCE_IP"

type OpenStackObject struct {
        Id       string `json:"id,omitempty"`
        Name     string `json:"name,omitempty"`
        TenantId string `json:"tenant_id,omitempty"`
}

type FixIP struct {
        SubnetId  string `json:"subnet_id"`
        IpAddress string `json:"ip_address"`
}

type OpenStackPort struct {
        OpenStackObject
        Status           string `json:"status,omitempty"`
        DnsName          string `json:"dns_name,omitempty"`
        NetworkId        string `json:"network_id,omitempty"`
        MacAddress       string `json:"mac_address,omitempty"`
        FixedIPS         []FixIP `json:"fixed_ips,omitempty"`
        AdminStateUp     bool `json:"admin_state_up,omitempty"`
        SecurityGroupIds []string `json:"security_groups,omitempty"`
        BindingHost      string `json:"binding:host_id,omitempty"`
        DeviceOwner      string `json:"device_owner,omitempty"`
        DeviceId         string `json:"device_id,omitempty"`
}

type PortsResponse struct {
        Ports []OpenStackPort `json:"ports"`
}

type OpenStackSubnet struct {
        OpenStackObject
        NetworkId string `json:"network_id"`
        GatewayIp string `json:"gateway_ip"`
        Cidr      string `json:"cidr"`
}
type SubnetResponse struct {
        Subnet OpenStackSubnet `json:"subnet"`
}

type LbObject struct {
        ProvisionStatus string `json:"provisioning_status,omitempty"`
        OperatingStatus string `json:"operating_status,omitempty"`
}

type LoadBalancer struct {
        OpenStackObject
        LbObject
        VipAddress  string `json:"vip_address,omitempty"`
        VipSubnetId string `json:"vip_subnet_id,omitempty"`
        Listeners   []Listener  `json:"listeners,omitempty"`
        Description string `json:"description,omitempty"`
        Pools       []Pool `json:"pools,omitempty"`
}

type LoadBalancersResponse struct {
        Lbs []LoadBalancer `json:"loadbalancers"`
}

type LoadBalancerResponse struct {
        LoadBalanacerBody LoadBalancer `json:"loadbalancer"`
}

type Member struct {
        OpenStackObject
        LbObject
        Address      string `json:"address"`
        ProtocolPort int32 `json:"protocol_port"`
        SubnetId     string `json:"subnet_id"`
        PoolId       string `json:"pool_id,omitempty"`
        Weight       int `json:"weight"`
}

type MemberResponse struct {
        MemberBody Member `json:"member"`
}

type SesssionPersistence struct {
        Type string `json:"type,omitempty"`
}

type Pool struct {
        OpenStackObject
        LbObject
        Members            []Member `json:"members,omitempty"`
        LbAlgorithm        string `json:"lb_algorithm,omitempty"`
        SessionPersistence SesssionPersistence `json:"session_persistence,omitempty"`
        ListenerId         string `json:"listener_id,omitempty"`
        LoadBalancerId     string `json:"loadbalancer_id,omitempty"`
        Protocol           string `json:"protocol"`
}

type PoolResponse struct {
        PoolBody Pool `json:"pool,omitempty"`
}

type Vip struct {
        OpenStackObject
}

type Listener struct {
        OpenStackObject
        LbObject
        Pools          []Pool `json:"pools,omitempty"`
        ProtocolPort   int32 `json:"protocol_port"`
        Protocol       string `json:"protocol"`
        LoadBalancerId string `json:"loadbalancer_id"`
}

type ListenerResponse struct {
        ListenerBody Listener `json:"listener"`
}

type LoadBalancerStatusResponse struct {
        Statuses LbStatusWrapper `json:"statuses"`
}

type LbStatusWrapper struct {
        Lb LoadBalancer `json:"loadbalancer"`
}
