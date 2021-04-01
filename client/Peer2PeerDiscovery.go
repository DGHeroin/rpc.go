package client

type (
    Peer2PeerDiscovery struct {
        server string
    }
)


func NewPeer2PeerDiscovery(server string) (Discovery, error) {
    return &Peer2PeerDiscovery{server: server}, nil
}


func (p *Peer2PeerDiscovery) GetServices() KVPairs {
    return []KVPair{
        {
            Key:   p.server,
            Value: "",
        },
    }
}
