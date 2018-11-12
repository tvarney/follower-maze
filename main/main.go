
package main

import (
    server "github.com/tvarney/follower-maze/serverlib"
)

func main() {
    ns := server.NewNetServer()
    ns.Run()
}

