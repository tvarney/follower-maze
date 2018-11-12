package net

import (
    "bufio"
    "expvar"
    "fmt"
    inet "net"
    "net/http"
    server "github.com/tvarney/follower-maze/serverlib"
    "strings"
    "sync"
)

var (
    cacheSize = expvar.NewInt("CachedMessages")
    messagesSent = expvar.NewInt("MessagesSent")
    messagesRecieved = expvar.NewInt("MessagesRecieved")
)

// NetClient is a network connected implementation of the Client interface.
type NetClient struct {
    conn inet.Conn
    userid server.UserId
}

func (nc *NetClient) Init() {
    reader := bufio.NewReader(nc.conn)
    idstr, err := reader.ReadString('\n')
    if err != nil {
        nc.userid = -1
        return
    }
    idstr = strings.TrimSpace(idstr)

    userid, err := server.ParseUserId(idstr)
    if err != nil {
        nc.userid = -1
    } else {
        nc.userid = userid
    }
}

func (nc *NetClient) Id() server.UserId {
    return nc.userid
}

func (nc *NetClient) WriteMessage(msg server.Message) {
    nc.conn.Write([]byte(msg.String()))
}

type NetSource struct {
    conn inet.Conn
    br *bufio.Reader
}

func (ns *NetSource) WaitMessage() (server.Message, error) {
    if ns.br == nil {
        ns.br = bufio.NewReader(ns.conn)
    }
    s, err := ns.br.ReadString('\n')
    msg, msgerr := server.ParseMessage(s)

    if err != nil {
        return msg, err
    }
    if msgerr != nil {
        return msg, msgerr
    }
    return msg, nil
}

type NetServer struct {
    // Data variables

    clientMux sync.Mutex
    clients map[server.UserId]server.Client
    source server.EventSource

    // Config variables
    // clientPort is the port on which the server should listen for clients
    clientPort uint64
    // eventSourcePort is the port to listen for EventSource connections
    eventSourcePort uint64
    // chanBuffer is the size of the channel buffers
    chanBuffer uint32

    // Followers is a map to follower sets.
    // Followers is part of the server implementation because the server needs
    // to keep track of who has followed who, regardless of if the Client has
    // connected.
    followers map[server.UserId]map[server.UserId]bool
    // acceptConnections controls the registration threads
    acceptConnections bool
    errors chan error
}

func NewNetServer() *NetServer {
    return &NetServer{
        clientPort: 9099,
        eventSourcePort: 9090,
        chanBuffer: 10,
        acceptConnections: false,
        source: nil,
        clients: make(map[server.UserId]server.Client),
        followers: map[server.UserId]map[server.UserId]bool{},
    }
}

func (ns *NetServer) RegisterClient(c server.Client) bool {
    // Call the client specific initialization hook. This may block, so
    // it may be best to launch this as a goroutine
    c.Init()
    ns.clientMux.Lock()
    _, ok := ns.clients[c.Id()]
    if ok {
        return false
    }
    ns.clients[c.Id()] = c
    ns.clientMux.Unlock()
    return true
}

func (ns *NetServer) UnregisterClient(c server.Client) bool {
    ns.clientMux.Lock()
    delete(ns.clients, c.Id())
    ns.clientMux.Unlock()
    return true
}

func (ns *NetServer) RegisterEventSource(es server.EventSource) bool {
    if ns.source != nil {
        return false
    }
    ns.source = es
    go ns.readSource(es)
    return true
}

func (ns *NetServer) UnregisterEventSource(es server.EventSource) bool {
    if ns.source == es {
        ns.source = nil
        return true
    }
    return false
}

func (ns *NetServer) Broadcast(msg server.Message) {
    for _, c := range ns.clients {
        c.WriteMessage(msg)
    }
}

func (ns *NetServer) GetClient(clientId server.UserId) server.Client {
    c, ok := ns.clients[clientId]
    if ok {
        return c
    }
    return nil
}

func (ns *NetServer) AddFollower(cid server.UserId, uid server.UserId) {
    fset, ok := ns.followers[cid]
    if !ok {
        fset = map[server.UserId]bool{}
        ns.followers[cid] = fset
    }

    fset[uid] = true
}

func (ns *NetServer) RemoveFollower(cid server.UserId, uid server.UserId) {
    set, ok := ns.followers[cid]
    if !ok {
        return
    }

    delete(set, uid)

    if len(set) == 0 {
        delete(ns.followers, cid)
    }
}

func (ns *NetServer) GetFollowers(uid server.UserId) map[server.UserId]bool {
    return ns.followers[uid]
}

func (ns *NetServer) Init() {
    if ns.errors == nil {
        ns.errors = make(chan error, ns.chanBuffer)
    }
}

// Run starts the server instance
func (ns *NetServer) Run() {
    ns.Init()

    if !ns.acceptConnections {
        fmt.Println("Starting registration threads...")
        ns.StartRegistrationThreads()
    }

    socket, err := inet.Listen("tcp", "localhost:8080")
    if err == nil {
        fmt.Println("expvar server on localhost:8080")
        http.Serve(socket, nil)
    } else {
        fmt.Println("Failed to start expvar server: ", err)
    }

    for {
        e := <-ns.errors
        fmt.Println("Error: ", e)
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
    go ns.acceptClients()
    go ns.acceptEventSource()
}

func (ns *NetServer) acceptClients() {
    ln, err := inet.Listen("tcp", fmt.Sprintf(":%d", ns.clientPort))
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
            go ns.RegisterClient(&c)
        }
    }
}

func (ns *NetServer) acceptEventSource() {
    ln, err := inet.Listen("tcp", fmt.Sprintf(":%d", ns.eventSourcePort))
    if err != nil {
        ns.errors <- err
        return
    }

    for ns.acceptConnections {
        conn, err := ln.Accept()
        if err != nil {
            ns.errors <- err
        } else {
            source := NetSource{
                conn: conn,
            }
            go ns.RegisterEventSource(&source)
        }
    }
}

func (ns *NetServer) readSource(source server.EventSource) {
    fmt.Println("Reading from event source...")
    current := server.SeqId(1)
    cache := make(map[server.SeqId]server.Message)

    for {
        msg, err := source.WaitMessage()

        if msg != nil {
            messagesRecieved.Add(1)
            if msg.Id() != current {
                cache[msg.Id()] = msg
                cacheSize.Set(int64(len(cache)))
            }else {
                // Pump from cache until mismatch again
                msg.Dispatch(ns)
                messagesSent.Add(1)
                current++
                msg, ok := cache[current]
                for ok {
                    msg.Dispatch(ns)
                    delete(cache, current)
                    messagesSent.Add(1)
                    current++
                    msg, ok = cache[current]
                }
                cacheSize.Set(int64(len(cache)))
            }
        } else {
            fmt.Println("Message is nil!")
        }

        if err != nil {
            fmt.Println("Error reading event source: ", err)
            ns.UnregisterEventSource(source)
            return
        }
    }
}

