
package serverlib

import (
    "errors"
    "fmt"
    "strings"
)


// Message is an interface used by the server to consume arbitrary messages.
type Message interface {
    // Id returns the sequence identifier of the message.
    Id() SeqId

    // String converts the message struct to a string writeable to a client/server.
    String() string

    // Dispatch takes a server argument and attempts to write itself to all applicable clients.
    Dispatch(Server) error
}

type Follow struct {
    Seq SeqId
    IdFrom UserId
    IdTo UserId
}

func NewFollow(s SeqId, args ...UserId) (Message, error) {
    if len(args) != 2 {
        return nil, errors.New("Invalid number of arguments to NewFollow")
    }
    return &Follow{
        Seq: s,
        IdFrom: args[0],
        IdTo: args[1],
    }, nil
}

func (f *Follow) Id() SeqId {
    return f.Seq
}

func (f *Follow) String() string {
    return fmt.Sprintf("%d|F|%d|%d\r\n", f.Seq, f.IdFrom, f.IdTo)
}

func (f *Follow) Dispatch(s Server) error {
    s.AddFollower(f.IdTo, f.IdFrom)
    c := s.GetClient(f.IdTo)
    if c != nil {
        c.WriteMessage(f)
    }
    return nil
}

type Unfollow struct {
    Seq SeqId
    IdFrom UserId
    IdTo UserId
}

func NewUnfollow(s SeqId, args ...UserId) (Message, error) {
    if len(args) != 2 {
        return nil, errors.New("Invalid number of arguments to NewUnfollow")
    }
    return &Unfollow{
        Seq: s,
        IdFrom: args[0],
        IdTo: args[1],
    }, nil
}

func (u *Unfollow) Id() SeqId {
    return u.Seq
}

func (u *Unfollow) String() string {
    return fmt.Sprintf("%d|U|%d|%d\r\n", u.Seq, u.IdFrom, u.IdTo)
}

func (u *Unfollow) Dispatch(server Server) error {
    server.RemoveFollower(u.IdTo, u.IdFrom)
    return nil
}

type Broadcast struct {
    Seq SeqId
}

func NewBroadcast(s SeqId, args ...UserId) (Message, error) {
    if len(args) != 0 {
        return nil, errors.New("Invalid number of arguments to NewBroadcast")
    }
    return &Broadcast{
        Seq: s,
    }, nil
}

func (b *Broadcast) Id() SeqId {
    return b.Seq
}

func (b *Broadcast) String() string {
    return fmt.Sprintf("%d|B\r\n", b.Seq)
}

func (b *Broadcast) Dispatch(server Server) error {
    server.Broadcast(b)
    return nil
}

type StatusUpdate struct {
    Seq SeqId
    IdFrom UserId
}

func NewStatusUpdate(s SeqId, args ...UserId) (Message, error) {
    if len(args) != 1 {
        return nil, errors.New("Invalid number of arguments to NewStatusUpdate")
    }
    return &StatusUpdate{
        Seq: s,
        IdFrom: args[0],
    }, nil
}

func (su *StatusUpdate) Id() SeqId {
    return su.Seq
}

func (su *StatusUpdate) String() string {
    return fmt.Sprintf("%d|S|%d\r\n", su.Seq, su.IdFrom)
}

func (su *StatusUpdate) Dispatch(server Server) error {
    followers := server.GetFollowers(su.IdFrom)
    if followers == nil {
        return nil
    }

    for uid, _ := range followers {
        c := server.GetClient(uid)
        if c != nil {
            c.WriteMessage(su)
        }
    }
    return nil
}

type PrivateMessage struct {
    Seq SeqId
    IdFrom UserId
    IdTo UserId
}

func NewPrivateMessage(s SeqId, args ...UserId) (Message, error) {
    if len(args) != 2 {
        return nil, errors.New("Invalid number of arguments to NewPrivateMessage")
    }
    return &PrivateMessage{
        Seq: s,
        IdFrom: args[0],
        IdTo: args[1],
    }, nil
}

func (pm *PrivateMessage) Id() SeqId {
    return pm.Seq
}

func (pm *PrivateMessage) String() string {
    return fmt.Sprintf("%d|P|%d|%d\r\n", pm.Seq, pm.IdFrom, pm.IdTo)
}

func (pm *PrivateMessage) Dispatch(server Server) error {
    c := server.GetClient(pm.IdTo)
    if c != nil {
        c.WriteMessage(pm)
    }
    return nil
}

// TODO: Make the function type a bit more general
var msgTypeMap map[string]func(SeqId, ...UserId)(Message, error) = map[string]func(SeqId, ...UserId)(Message, error){
    "F": NewFollow,
    "U": NewUnfollow,
    "B": NewBroadcast,
    "S": NewStatusUpdate,
    "P": NewPrivateMessage,
}

// NewMessage creates a new message of the appropriate type
func NewMessage(seqNo SeqId, msgType string, args ...UserId) (Message, error) {
    fnew, ok := msgTypeMap[msgType]
    if ok {
        return fnew(seqNo, args...)
    } else {
        return nil, errors.New("Unknown message type " + msgType)
    }
}

func ParseMessage(msgStr string) (Message, error) {
    msgStr = strings.TrimSpace(msgStr)
    parts := strings.Split(msgStr, "|")
    if len(parts) < 2 {
        return nil, errors.New("Invalid message format: " + msgStr)
    }
    seq, err := ParseSeqId(parts[0])
    if err != nil {
        return nil, err
    }

    msgType := parts[1]

    var args []UserId
    for _, arg := range parts[2:] {
        v, err := ParseUserId(arg)
        if err != nil {
            return nil, err
        }
        args = append(args, v)
    }

    return NewMessage(SeqId(seq), msgType, args...)
}

