package rpc

type PluginContainer struct {
    plugins []interface{}
}

func (c *PluginContainer) Add(p interface{}) {
    c.plugins = append(c.plugins, p)
}

func (c *PluginContainer) Remove(p interface{}) {
    var plugins []interface{}
    for _, v := range c.plugins {
        if v == p {
            continue
        }
        plugins = append(plugins, v)
    }
    c.plugins = plugins
}
func (c *PluginContainer) Range(fn func(interface{})) {
    for _, v := range c.plugins {
        fn(v)
    }
}
