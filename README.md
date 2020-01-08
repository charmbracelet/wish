# Wish

Command based SSH server for building apps.

## How It Works

Wish is similar to web frameworks in that it offers routing and structure
around SSH. Instead of URL based routing, Wish uses commands issued to the
`ssh` client to call handlers.

## Server Example

```
package main

import (
        "fmt"
        "strings"

        "github.com/charmbracelet/wish"
)

func echoHandler(s wish.Session) {
        var out string
        cmd := s.Command()
        if len(cmd) > 1 {
                out = strings.join(cmd[1:], " ")
        } else {
                out = "Usage: ssh HOST echo SOME STUFF"
        }
        s.Write([]byte(fmt.Sprintf("\n\n%s\n", out)))
}

func main() {
        port := 5555
        keyPath := "./.ssh/id_rsa"
        server := wish.NewServer(keyPath, port)
        server.addHandler("echo", echoHandler)
        server.Start()
}
```

## Client Example

```
$ ssh -p 5555 host echo this is my wish

this is my wish
```
