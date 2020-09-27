package rpc

import (
    "bytes"
    "encoding/binary"
    "errors"
    "io"
)

var (
    ErrorMessageFormatInvalid = errors.New("message format invalid")
    ErrorConnectIdInvalid     = errors.New("connect id invalid")
    ErrorConnectionInvalid    = errors.New("connection invalid")
)

type MessageType uint8

const (
    MessageTypeKeep  = MessageType(1)
    MessageTypeData  = MessageType(2)
    MessageTypeReply = MessageType(3)
)

type Message struct {
    Type      MessageType
    Payload   []byte
    Tag       uint32
    requestId uint32
    conn      io.Writer
}

func (m *Message) Reply(tag uint32, payload []byte) {
    go writeMessage(m.conn, &Message{
        Type:      MessageTypeReply,
        requestId: m.requestId,
        Tag:       tag,
        Payload:   payload,
    })
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
func readMessage(conn io.ReadWriter) (*Message, error) {
    message := &Message{
        conn: conn,
    }
    // read header
    header := make([]byte, 1)
    if _, err := io.ReadFull(conn, header); err != nil {
        return nil, err
    }
    msgType := MessageType(header[0])

    if msgType == MessageTypeKeep {
        return nil, nil
    }
    // msg type
    message.Type = msgType
    // size
    size, err := readUInt32(conn)
    if err != nil {
        return nil, err
    }
    // tag
    tag, err := readUInt32(conn)
    if err != nil {
        return nil, err
    }
    // request id
    requestId, err := readUInt32(conn)
    if err != nil {
        return nil, err
    }
    // payload
    payload := make([]byte, size)
    if _, err := io.ReadFull(conn, payload); err != nil {
        return nil, err
    }
    message.requestId = requestId
    message.Type = msgType
    message.Tag = tag
    message.Payload = payload
    return message, nil

}

func writeMessage(conn io.Writer, message *Message) (n int, err error) {
    buffer := bytes.NewBuffer([]byte{uint8(message.Type)}) // msg type
    if message.Type == MessageTypeKeep {
    } else {
        writeUInt32(uint32(len(message.Payload)), buffer) // size
        writeUInt32(message.Tag, buffer)                  // tag
        writeUInt32(message.requestId, buffer)            // request id
        buffer.Write(message.Payload)
    }

    return conn.Write(buffer.Bytes())
}
