package rpc

import (
    "bytes"
    "encoding/binary"
    "errors"
    "io"
    "sync"
)

var (
    ErrorMessageFormatInvalid = errors.New("message format invalid")
    ErrorConnectIdInvalid     = errors.New("connect id invalid")
    ErrorConnectionInvalid    = errors.New("connection invalid")
)

type MessageType uint8

const (
    MessageTypeKeep     = MessageType(1) // 链接保持消息
    MessageTypeRequest  = MessageType(2) // 请求消息，必须有回复
    MessageTypeResponse = MessageType(3) // 请求消息的回复
    MessageTypeOneWay   = MessageType(4) // 单向消息，忽略回复
)

type (
    Message struct {
        Type      MessageType
        Payload   []byte
        Tag       uint32
        requestId uint32
        conn      io.Writer
    }
)

var (
    messagePool = sync.Pool{New: func() interface{} {
        return &Message{}
    }}
)

func NewMessage() *Message {
    return messagePool.Get().(*Message)
}

func FreeMessage(message *Message) {
    message.Type = 0
    message.Tag = 0
    message.Payload = nil
    message.requestId = 0
    message.conn = nil
    messagePool.Put(message)
}

func (m *Message) Reply(tag uint32, payload []byte) {
    if m.Type != MessageTypeRequest {
        return
    }
    msg := NewMessage()
    msg.Tag = tag
    msg.Payload = payload
    msg.Type = MessageTypeResponse
    msg.requestId = m.requestId

    writeMessage(m.conn, msg)
}
func readUInt32(c io.Reader) (uint32, error) {
    data := make([]byte, 4)
    if _, err := io.ReadFull(c, data); err != nil {
        return 0, err
    }
    return binary.BigEndian.Uint32(data), nil
}
func writeUInt32(val uint32, buffer *bytes.Buffer) {
    data := make([]byte, 4)
    binary.BigEndian.PutUint32(data, val)
    buffer.Write(data)
}

func writeUInt32Bytes(val uint32, buffer []byte, offset int) {
    binary.BigEndian.PutUint32(buffer[offset:], val)
}

func readMessage(conn io.ReadWriter) (*Message, error) {
    message := NewMessage()
    message.conn = conn
    // read header
    header := make([]byte, 1)
    if _, err := io.ReadFull(conn, header); err != nil {
        FreeMessage(message)
        return nil, err
    }
    msgType := MessageType(header[0])

    if msgType == MessageTypeKeep {
        FreeMessage(message)
        return nil, nil
    }
    // msg type
    message.Type = msgType
    // size
    size, err := readUInt32(conn)
    if err != nil {
        FreeMessage(message)
        return nil, err
    }
    // tag
    tag, err := readUInt32(conn)
    if err != nil {
        FreeMessage(message)
        return nil, err
    }
    // request id
    requestId, err := readUInt32(conn)
    if err != nil {
        FreeMessage(message)
        return nil, err
    }
    // payload
    payload := make([]byte, size)
    if _, err2 := io.ReadFull(conn, payload); err2 != nil {
        FreeMessage(message)
        return nil, err2
    }
    message.requestId = requestId
    message.Type = msgType
    message.Tag = tag
    message.Payload = payload
    return message, nil
}

func writeMessage(conn io.Writer, mess ...*Message) (n int, err error) {
    allBuff := bytes.NewBuffer(nil)
    for _, message := range mess {
        switch message.Type {
        case MessageTypeKeep:
            allBuff.Write([]byte{uint8(message.Type)})
        default:
            buffer := bytes.NewBuffer(nil)
            buffer.Write([]byte{uint8(message.Type)})         // msg type  1
            writeUInt32(uint32(len(message.Payload)), buffer) // size  4
            writeUInt32(message.Tag, buffer)                  // tag 4
            writeUInt32(message.requestId, buffer)            // request id 4
            buffer.Write(message.Payload)
            allBuff.Write(buffer.Bytes())
        }
        FreeMessage(message)
    }
    return conn.Write(allBuff.Bytes())
}
