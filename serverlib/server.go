
package serverlib

// Client is an implementation of a writable client connection.
// The Client interface must provide a uint64 id and a method to write messages
// to it. The Init() method is provided for the server to do any post creation
// initialization. The NetClient structure uses this to read the user-id of
// the client from the connection.
type Client interface {
    Init()
    Id() uint64
    Write(*Message)
}

// EventSource is the interface used to read events from an arbitrary source.
type EventSource interface {
    WaitMessage() *Message
}

// Server is the interface expected of the follower maze server implementation.
// The Server interface exposes methods for manipulating and running an
// arbitrary server implementation.
type Server interface {
    RegisterClient(*Client)
    UnregisterClient(*Client)
    RegisterEventSource(*EventSource)
    UnregisterEventSource(*EventSource)

    GetClient(uint64) *Client

    Run()
}

type NetServer struct {
    clientPort uint64
    eventSourcePort uint64
}

func NewNetServer() *NetServer {
    ns := NetServer{
        clientPort: 9090,
        eventSourcePort: 9099,
    }
    return &ns
}

func (ns *NetServer) RegisterClient(c *Client) {

}

func (ns *NetServer) UnregisterClient(c *Client) {

}

func (ns *NetServer) RegisterEventSource(c *Client) {

}

func (ns *NetServer) UnregisterEventSource(c *Client) {

}

func (ns *NetServer) GetClient(clientId uint64) *Client {
    return nil
}

func (ns *NetServer) GetClientPort() uint64 {
    return ns.clientPort
}

func (ns *NetServer) SetClientPort(port uint64) {
    ns.clientPort = port
}

func (ns *NetServer) GetEventSourcePort() uint64 {
    return ns.eventSourcePort
}

func (ns *NetServer) SetEventSourcePort(port uint64) {
    ns.eventSourcePort = port
}

