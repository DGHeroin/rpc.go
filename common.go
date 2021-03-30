package rpc

import (
    "bytes"
    "encoding/binary"
    "errors"
    "io"
)

var (
    ErrorMessageFormatInvalid = errors.New("message format invalid")
    ErrorMessageTypeInvalid   = errors.New("message type invalid")
    ErrorConnectionInvalid    = errors.New("connection invalid")
)

type MessageType uint8

const (
    MessageTypeKeep     = MessageType(1) // 链接保持消息
    MessageTypeRequest  = MessageType(2) // 请求消息，必须有回复
    MessageTypeResponse = MessageType(3) // 请求消息的回复
    MessageTypeOneWay   = MessageType(4) // 单向消息，忽略回复
    MessageTypeClose    = MessageType(5) // 关闭消息
)

type (
    Message struct {
        Type      MessageType
        Payload   []byte
        requestId uint32
        SendCh    chan []byte
    }
)

func NewMessage(ch chan []byte) *Message {
    return &Message{
        SendCh: ch,
    }
}

func (m *Message) Reply(payload []byte) error {
    if m.Type != MessageTypeRequest {
        return ErrorMessageTypeInvalid
    }
    msg := NewMessage(m.SendCh)
    msg.Payload = payload
    msg.Type = MessageTypeResponse
    msg.requestId = m.requestId
    msg.Emit()
    return nil
}

func (m *Message) Emit() {
    bin := m.Encode()
    m.SendCh <- bin
}

func readFull(r io.Reader, data []byte) (int, error) {
    n, err := io.ReadFull(r, data)
    return n, err
}

func (m *Message) Decode(conn io.ReadWriteCloser) error {
    var (
        err  error
        size uint32
    )

    // read header
    header := make([]byte, 1)
    if _, err = readFull(conn, header); err != nil {
        return err
    }
    m.Type = MessageType(header[0])
    if m.Type == MessageTypeKeep {
        return nil
    }
    // size
    size, err = readUInt32(conn)
    if err != nil {
        return err
    }
    switch m.Type {
    case MessageTypeRequest, MessageTypeResponse:
        // request id
        m.requestId, err = readUInt32(conn)
        if err != nil {
            return err
        }
    }
    // payload
    m.Payload = make([]byte, size)
    _, err = readFull(conn, m.Payload)
    if err != nil {
        return err
    }
    return nil
}

func (m *Message) Encode() []byte {
    if m.Type == MessageTypeKeep {
        return []byte{uint8(m.Type)}
    }
    buffer := bytes.NewBuffer(nil)
    buffer.Write([]byte{uint8(m.Type)})         // msg type  1
    writeUInt32(uint32(len(m.Payload)), buffer) // size  4
    switch m.Type {
    case MessageTypeRequest, MessageTypeResponse:
        writeUInt32(m.requestId, buffer) // request id 4
    }
    buffer.Write(m.Payload)
    return buffer.Bytes()
}

func readUInt32(c io.Reader) (uint32, error) {
    data := make([]byte, 4)
    if _, err := readFull(c, data); err != nil {
        return 0, err
    }
    return binary.BigEndian.Uint32(data), nil
}
func writeUInt32(val uint32, buffer *bytes.Buffer) {
    data := make([]byte, 4)
    binary.BigEndian.PutUint32(data, val)
    buffer.Write(data)
}
