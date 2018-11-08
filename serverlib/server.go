
package serverlib

import (
    "bufio"
    "fmt"
    "net"
    //"runtime"
    "strconv"
    "strings"
    "sync"
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
    return UserId(idno), err
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
    Init()
    Id() UserId
    Write(Message)
}

// EventSource is the interface used to read events from an arbitrary source.
type EventSource interface {
    WaitMessage() Message
}

// Server is the interface expected of the follower maze server implementation.
// The Server interface exposes methods for manipulating and running an
// arbitrary server implementation.
type Server interface {
    RegisterClient(Client)
    UnregisterClient(Client)
    RegisterEventSource(EventSource)
    UnregisterEventSource(EventSource)

    GetClient(uint64) Client

    Init()
    Run()
}

type NetClient struct {
    conn net.Conn
    userid UserId
}

func (nc *NetClient) Init() {
    reader := bufio.NewReader(nc.conn)
    idstr, err := reader.ReadString('\n')
    if err != nil {
        nc.userid = -1
        return
    }
    idstr = strings.TrimSpace(idstr)

    userid, err := ParseUserId(idstr)
    if err != nil {
        nc.userid = -1
    } else {
        nc.userid = userid
    }
}

func (nc *NetClient) Id() UserId {
    return nc.userid
}

func (nc *NetClient) Write(msg Message) {
    nc.conn.Write([]byte(msg.String()))
}

type NetServer struct {
    // Data variables

    clientMux sync.Mutex
    clients map[UserId]Client
    sourceMux sync.Mutex
    sources map[EventSource]bool

    // Config variables

    // clientPort is the port on which the server should listen for clients
    clientPort uint64
    // eventSourcePort is the port to listen for EventSource connections
    eventSourcePort uint64
    // chanBuffer is the size of the channel buffers
    chanBuffer uint32

    // Runtime variables

    // acceptConnections controls the registration threads
    acceptConnections bool
    // regClient is a channel for registering clients
    regClient chan Client
    unregClient chan Client
    regEventSource chan EventSource
    unregEventSource chan EventSource

    errors chan error
}

func NewNetServer() *NetServer {
    ns := NetServer{
        clientPort: 9090,
        eventSourcePort: 9099,
        chanBuffer: 10,
        acceptConnections: false,
    }
    return &ns
}

func (ns *NetServer) RegisterClient(c Client) {
    if ns.regClient != nil && ns.acceptConnections {
        ns.regClient <- c
    }
}

func (ns *NetServer) UnregisterClient(c Client) {
    if ns.unregClient != nil && ns.acceptConnections {
        ns.unregClient <- c
    }
}

func (ns *NetServer) RegisterEventSource(es EventSource) {
    if ns.regEventSource != nil && ns.acceptConnections {
        ns.regEventSource <- es
    }
}

func (ns *NetServer) UnregisterEventSource(es EventSource) { 
    if ns.unregEventSource != nil && ns.acceptConnections {
        ns.unregEventSource <- es
    }
}

func (ns *NetServer) GetClient(clientId uint64) Client {
    return nil
}

func (ns *NetServer) Init() {
    ns.regClient = make(chan Client, ns.chanBuffer)
    ns.unregClient = make(chan Client, ns.chanBuffer)
    ns.regEventSource = make(chan EventSource, ns.chanBuffer)
    ns.unregEventSource = make(chan EventSource, ns.chanBuffer)

    if ns.errors == nil {
        ns.errors = make(chan error, ns.chanBuffer)
    }
}

func (ns *NetServer) Run() {
    if ns.regClient == nil {
        ns.Init()
    }

    if !ns.acceptConnections {
        ns.StartRegistrationThreads()
    }


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

// SetEventSourcePort changes the port that the server listens on for
// EventSources.
// If the Listener is already present, then this method does not change
// the port being listened on.
func (ns *NetServer) SetEventSourcePort(port uint64) {
    ns.eventSourcePort = port
}

// SetChanBufferSize attempts to set the channel buffer size.
// This method must be called before Init() or Run(), otherwise it will
// have no effect. If the channels have already been initialized, then this
// method will return false and not set the buffer size.
func (ns *NetServer) SetChanBufferSize(size uint32) bool {
    if ns.regClient == nil && ns.unregClient == nil &&
        ns.regEventSource == nil && ns.unregEventSource == nil {
        ns.chanBuffer = size
        return true
    }
    return false
}

// GetChanBufferSize returns the size used for channel buffering.
// This value is used for all 4 underlying channels
func (ns *NetServer) GetChanBufferSize() uint32 {
    return ns.chanBuffer
}

// StartRegistrationThreads launches a goroutine to handle the registration
// and unregistering of Clients and EventSources.
func (ns *NetServer) StartRegistrationThreads() {
    ns.acceptConnections = true
    // Launch the goroutines
    go ns.acceptRegClient()
    go ns.acceptUnregClient()
    go ns.acceptRegEventSource()
    go ns.acceptUnregEventSource()
}

func (ns *NetServer) acceptClients() {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", ns.clientPort))
    if err != nil {
        ns.errors <- err
        return
    }

    for ns.acceptConnections {
        conn, err := ln.Accept()
        if err != nil {
            ns.errors <- err
        } else {
            // Call the Register client with a new NetClient
            c := NetClient{
                conn: conn,
            }
            ns.RegisterClient(&c)
        }
    }
}

func (ns *NetServer) acceptRegClient() {
    var client Client
    var more bool
    for ns.acceptConnections {
        client, more = <-ns.regClient
        if more {
            // Handle the new client
            ns.clientMux.Lock()
            ns.clients[client.Id()] = client
            ns.clientMux.Unlock()
        } else {
            break
        }
    }
}

func (ns *NetServer) acceptUnregClient() {
    var client Client
    var more bool
    for ns.acceptConnections {
        client, more = <-ns.unregClient
        if more {
            // Handle removing the client
            ns.clientMux.Lock()
            delete(ns.clients, client.Id())
            ns.clientMux.Unlock()
        } else {
            // more is false, so the channel has been closed
            break
        }
    }
}

func (ns *NetServer) acceptRegEventSource() {
    var source EventSource
    var more bool
    for ns.acceptConnections {
        source, more = <-ns.regEventSource
        if more {
            // Handle adding the new event source
            ns.sourceMux.Lock()
            ns.sources[source] = true
            ns.sourceMux.Unlock()
        } else {
            // more is false, so the channel has been closed
            break
        }
    }
}

func (ns *NetServer) acceptUnregEventSource() {
    var source EventSource
    var more bool
    for ns.acceptConnections {
        source, more = <-ns.regEventSource
        if more {
            // Handle removing the event source
            ns.sourceMux.Lock()
            delete(ns.sources, source)
            ns.sourceMux.Unlock()
        } else {
            break
        }
    }
}

func (ns *NetServer) readSource(source EventSource) {
}

