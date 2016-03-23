# webbrowser

Go library for opening URLs using a web browser. Currently supports Linux, OSX, and Windows.

## Documentation

The API documentation can be found at [http://godoc.org/github.com/bryanl/webbrowser](http://godoc.org/github.com/bryanl/webbrowser).

## Example

```
package main

import (
    "log"

    "github.com/bryanl/webbrowser"
)

func main() {
    err := webbrowser.Open("http://blil.es", webbrowser.NewTab, true)
    if err != nil {
        log.Println("open failed:", err)
    }
}
```
~
