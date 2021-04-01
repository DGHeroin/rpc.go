package client

type (
    Discovery interface {
        GetServices() KVPairs
    }
    KVPair struct {
        Key   string
        Value string
    }
    KVPairs []KVPair
)

func (kv KVPairs) ToMap() map[string]string {
    result := make(map[string]string, len(kv))
    for _, v := range kv {
        result[v.Key] = v.Value
    }
    return result
}

func (kv KVPairs) Keys() []string {
    result := make([]string, len(kv))
    for _, v := range kv {
        result = append(result, v.Key)
    }
    return result
}

func (kv KVPairs) Values() []string {
    result := make([]string, len(kv))
    for _, v := range kv {
        result = append(result, v.Value)
    }
    return result
}
