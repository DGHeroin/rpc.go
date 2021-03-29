package rpc

// server side
type (
    ServerOnAcceptPlugin interface {
        OnAccept(id uint64)
    }
    ServerOnClosePlugin interface {
        OnClose(id uint64)
    }
    ServerOnMessagePlugin interface {
        OnMessage(id uint64, msg *Message)
    }
)

// client side
type (
    ClientOnOpenPlugin interface {
        OnOpen()
    }
    ClientOnClosePlugin interface {
        OnClose()
    }
    ClientOnMessagePlugin interface {
        OnMessage(msg *Message)
    }
)
