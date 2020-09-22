package rpc

import (
	"encoding/binary"
	"errors"
	"io"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

func init()  {
	logrus.WithField("stack", string(debug.Stack()))
}

type Event struct {
	ClientId uint64
	RequestId uint32
	Request  *Request
}

var (
	ErrorMessageFormatInvalid = errors.New("message format invalid")
	ErrorConnectIdInvalid     = errors.New("connect id invalid")
	ErrorConnectionInvalid    = errors.New("connection invalid")
)
type MessageType uint8
const (
	MessageTypeData = MessageType(iota)
	MessageTypePing
	MessageTypePong
)

func readMessage(conn io.ReadWriter) (MessageType, uint32, []byte, error) {
	// read header
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}

	msgType := MessageType(header[0])
	size := binary.BigEndian.Uint32(header[1:])

	switch msgType {
	case MessageTypeData: // data
		tagBytes := make([]byte, 4)
		if _, err := io.ReadFull(conn, tagBytes); err != nil {
			return msgType, 0, nil, err
		}
		tag := binary.BigEndian.Uint32(tagBytes)
		payload := make([]byte, size)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return msgType, 0, nil, err
		}
		return msgType, tag, payload, nil
	case MessageTypePing: // ping
		// response pong
		writeMessage(conn, 2, 0, nil)
		return msgType, 0, nil, nil
	case MessageTypePong: // pong
		return msgType, 0,nil, nil
	default:
		// error
		return msgType, 0, nil, ErrorMessageFormatInvalid
	}
}


func writeMessage(conn io.ReadWriter, msgType MessageType, tag uint32, data []byte) (n int, err error) {
	header := make([]byte, 5)
	header[0] = uint8(msgType)
	binary.BigEndian.PutUint32(header[1:], uint32(len(data)))
	if msgType == MessageTypeData {
		tagBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(tagBytes, tag)
		header = append(header, tagBytes...)
	}
	headerAndPayload := append(header, data...)
	return conn.Write(headerAndPayload)
}