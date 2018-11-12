
package serverlib

import (
    "strconv"
)

type UserId int64
type SeqId uint64

const (
    UserIdInvalid = UserId(-1)
)

func ParseUserId(data string) (UserId, error) {
    idno, err := strconv.ParseInt(data, 10, 64)
    if err != nil {
        return UserIdInvalid, err
    }
    return UserId(idno), nil
}

func ParseSeqId(data string) (SeqId, error) {
    idno, err := strconv.ParseUint(data, 10, 64)
    return SeqId(idno), err
}

// Client is an implementation of a writable client connection.
// The Client interface must provide a uint64 id and a method to write messages
// to it. The Init() method is provided for the server to do any post creation
// initialization. The NetClient structure uses this to read the user-id of
// the client from the connection.
type Client interface {
    // Init does any client specific initialization.
    // This should be called by the server implementation when the client is
    // registered.
    Init()

    // Id returns the unique UserId of the Client
    Id() UserId

    // WriteMessage takes an implementation of a Message and writes it to the underlying client implementation.
    WriteMessage(Message)
}

// EventSource is the interface used to read events from an arbitrary source.
type EventSource interface {
    WaitMessage() (Message, error)
}

// Server is the interface expected of the follower maze server implementation.
// The Server interface exposes methods for manipulating and running an
// arbitrary server implementation.
type Server interface {
    // RegisterClient adds a Client to the server
    RegisterClient(Client) bool

    // UnregisterClient removes a client from the server, if it exists
    UnregisterClient(Client) bool

    // RegisterEventSource adds an EventSource to the server
    // This method is not guaranteed to work for more than one event source at
    // a time.
    RegisterEventSource(EventSource) bool

    // UnregisterEventSource removes an EventSource from the server
    UnregisterEventSource(EventSource) bool

    // GetClient returns the client identified by the given UserId, or Nil of the client isn't registered to
    // this server.
    GetClient(UserId) Client

    // Broadcast sends the message to all attached clients, regardless of the dispatch implementation
    Broadcast(Message)

    // AddFollower tells the Server that a client has followed another
    AddFollower(UserId, UserId)

    // RemoveFollower tells the Server that a client has unfollowed another
    RemoveFollower(UserId, UserId)

    // GetFollowers returns a set (implemented as a map[UserId]bool) of all followers of the given client
    GetFollowers(UserId) map[UserId]bool

    // Init performs any Server specific initialization needed to run the server.
    // The Run() method should detect if this method has been called or not and call
    // it if it has not.
    Init()

    // Run starts the server process
    Run()
}

