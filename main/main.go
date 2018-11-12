
package main

import (
    server "github.com/tvarney/follower-maze/serverlib/net"
)

func main() {
    ns := server.NewNetServer()
    ns.Run()
}

