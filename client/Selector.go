package client

import (
    "context"
    "math/rand"
)

type Selector interface {
    Select(ctx context.Context, servicePath, serviceMethod string, args interface{}) string
}

type SelectMode int

const (
    RandomSelect SelectMode = iota
)

func newSelector(selectMode SelectMode, servers []string) Selector {
    switch selectMode {
    case RandomSelect:
        return newRandomSelect(servers)
    default:
        return newRandomSelect(servers)
    }
}

func newRandomSelect(servers []string) Selector {
    return &randomSelector{servers: servers}
}

// randomSelector selects randomly.
type randomSelector struct {
    servers []string
}

func newRandomSelector(servers map[string]string) Selector {
    ss := make([]string, 0, len(servers))
    for k := range servers {
        ss = append(ss, k)
    }

    return &randomSelector{servers: ss}
}
func (s randomSelector) Select(ctx context.Context, servicePath, serviceMethod string, args interface{}) string {
    ss := s.servers
    if len(ss) == 0 {
        return ""
    }
    return ss[rand.Intn(len(ss))]
}
