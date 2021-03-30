package kcp

import (
    "crypto/sha1"
    "github.com/xtaci/kcp-go"
    "golang.org/x/crypto/pbkdf2"
    "net"
)

func NewKCPDialer(address string, password []byte, salt []byte) (net.Conn, error) {
    if password != nil && salt != nil {
        key := pbkdf2.Key(password, salt, 1024, 32, sha1.New)
        if block, err := kcp.NewAESBlockCrypt(key); err != nil {
            return nil, err
        } else {
            return kcp.DialWithOptions(address, block, 10, 3)
        }
    } else {
        return kcp.Dial(address)
    }
}
