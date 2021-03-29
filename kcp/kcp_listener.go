package kcp

import (
    "crypto/sha1"
    "github.com/xtaci/kcp-go"
    "golang.org/x/crypto/pbkdf2"
    "net"
)

func NewKCPListenerServe(address string, password []byte, salt []byte) (net.Listener,error) {
    if password != nil && salt != nil {
        key := pbkdf2.Key(password, salt, 1024, 32, sha1.New)
        block, err := kcp.NewAESBlockCrypt(key)
        if err != nil {
            return nil, err
        }
        return kcp.ListenWithOptions(address, block, 10, 3)

    } else {
        return kcp.Listen(address)
    }
}