package client

import (
    "context"
    "github.com/DGHeroin/rpc.go"
    "golang.org/x/sync/singleflight"
    "log"
    "strings"
)

type (
    Option struct {
        Retries    int
        SelectMode SelectMode
    }
    Client interface {
        Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error
    }
    RPCClient interface {
    }
    xClient struct {
        servicePath  string
        selector     Selector
        Plugins      rpc.PluginContainer
        cachedClient map[string]RPCClient
    }
)

var (
    DefaultOption = Option{
        Retries: 3,
    }
)

func NewClient(servicePath string, discovery Discovery, option Option) Client {
    client := &xClient{
        servicePath:  servicePath,
        cachedClient: make(map[string]RPCClient),
    }
    servers := discovery.GetServices().Keys()
    client.selector = newSelector(option.SelectMode, servers)
    return client
}

func (c *xClient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
    addr, cli, err := c.selectClient(ctx, serviceMethod, args)
    if err != nil {
        return err
    }
    log.Println(addr, cli)
    return nil
}

func (c *xClient) selectClient(ctx context.Context, serviceMethod string, args interface{}) (string, RPCClient, error) {
    var (
    	ok bool
        client RPCClient
    )
    selectedAddr := c.selector.Select(ctx, c.servicePath, serviceMethod, args)
    client, ok = c.cachedClient[selectedAddr]
    if !ok {
        network, addr := splitNetworkAndAddress(selectedAddr)
    }
    var sg  singleflight.Group{}
    sg.
    return selectedAddr, client, nil
}
func splitNetworkAndAddress(server string) (string, string) {
    ss := strings.SplitN(server, "@", 2)
    if len(ss) == 1 {
        return "tcp", server
    }

    return ss[0], ss[1]
}
