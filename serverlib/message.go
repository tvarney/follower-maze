
package serverlib

import (
    "errors"
    "fmt"
    "strconv"
    "strings"
)

type UserId uint64
type SeqId uint64

const (
    MsgTypeError uint8 = 0
    MsgFollow uint8 = 1
    MsgUnfollow uint8 = 2
    MsgBroadcast uint8 = 3
    MsgPrivateMsg uint8 = 4
    MsgStatusUpdate uint8 = 5
)

var typeStrMap map[string]uint8 = map[string]uint8{
    "F": MsgFollow,
    "U": MsgUnfollow,
    "B": MsgBroadcast,
    "P": MsgPrivateMsg,
    "S": MsgStatusUpdate,
}

var typeNumMap map[uint8]string = map[uint8]string{
    MsgFollow: "F",
    MsgUnfollow: "U",
    MsgBroadcast: "B",
    MsgPrivateMsg: "P",
    MsgStatusUpdate: "S",
}

// Message is an interface used by the server to consume arbitrary messages.
type Message interface {
    MsgType() uint8
    Id() SeqId
    Format() string
}

// MessageNoArgs is a Message with no arguments.
// This message type is used for the Broadcast message.
type MessageNoArgs struct {
    MType uint8
    Seq SeqId
}

func (m *MessageNoArgs) MsgType() uint8 {
    return m.MType
}

func (m *MessageNoArgs) Id() SeqId {
    return m.Seq
}

func (m *MessageNoArgs) Format() string {
    return fmt.Sprintf("%d\\|%s\r\n", m.Seq, MessageTypeStr(m.MType))
}

// MessageFrom is a Message with only a From argument.
// This message type is used for the StatusUpdate message.
type MessageFrom struct {
    MType uint8
    Seq SeqId
    IdFrom UserId
}

func (m *MessageFrom) MsgType() uint8 {
    return m.MType
}

func (m *MessageFrom) Id() SeqId {
    return m.Seq
}

func (m *MessageFrom) Format() string {
    return fmt.Sprintf("%d\\|%s\\|%d\r\n", m.Seq, MessageTypeStr(m.MType), m.IdFrom)
}

// MessageFromTo is a Message with both From and To arguments.
// This message type is used for the Follow, Unfollow, and PrivateMsg
// messages.
type MessageFromTo struct {
    MType uint8
    Seq SeqId
    IdFrom UserId
    IdTo UserId
}

func (m *MessageFromTo) MsgType() uint8 {
    return m.MType
}

func (m *MessageFromTo) Id() SeqId {
    return m.Seq
}

func (m *MessageFromTo) Format() string {
    return fmt.Sprintf("%d\\|%s\\|%d\\|%d\r\n", m.Seq, MessageTypeStr(m.MType), m.IdFrom, m.IdTo)
}

// NewMessage creates a new message of the appropriate type
func NewMessage(seqNo SeqId, msgType uint8, args ...UserId) (Message, error) {
    switch(msgType) {
    case MsgFollow:
        fallthrough
    case MsgUnfollow:
        fallthrough
    case MsgPrivateMsg:
        if len(args) != 2 {
            return nil, errors.New("Invalid number of arguments to message")
        }
        return &MessageFromTo{
            MType: msgType,
            Seq: seqNo,
            IdFrom: args[0],
            IdTo: args[1],
        }, nil
    case MsgBroadcast:
        if len(args) != 0 {
            return nil, errors.New("Invalid number of arguments to message")
        }
        return &MessageNoArgs{
            MType: msgType,
            Seq: seqNo,
        }, nil
    case MsgStatusUpdate:
        if len(args) != 1 {
            return nil, errors.New("Invalid number of arguments to message")
        }
        return &MessageFrom{
            MType: msgType,
            Seq: seqNo,
            IdFrom: args[0],
        }, nil
    }
    return nil, errors.New("Unknown message type")
}

func ParseMessage(msgStr string) (Message, error) {
    parts := strings.Split(msgStr, "\\|")
    if len(parts) <= 2 {
        return nil, errors.New("Invalid message format")
    }
    seq, err := strconv.ParseUint(parts[0], 10, 64)
    if err != nil {
        return nil, err
    }

    msgTypeNo := MessageTypeNo(parts[1])

    var args []UserId
    for _, arg := range parts[2:] {
        v, err := strconv.ParseUint(arg, 10, 64)
        if err != nil {
            return nil, err
        }
        args = append(args, UserId(v))
    }

    return NewMessage(SeqId(seq), msgTypeNo, args...)
}

func MessageTypeStr(msgType uint8) string {
    msgTypeChar, ok := typeNumMap[msgType]
    if !ok {
        return ""
    }
    return msgTypeChar
}

func MessageTypeNo(msgTypeChar string) uint8 {
    msgType, ok := typeStrMap[msgTypeChar]
    if !ok {
        return MsgTypeError
    }
    return msgType
}

