package main

import (
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"time"

	"github.com/natefinch/pie"
)

var max int = 2000
var path string = "./plugin.py"

type plug struct {
	Client *rpc.Client
}

func createClient() *plug {
	log.Printf("Creating plugin")
	client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, path)
	if err != nil {
		log.Printf("Create error: %v", err)
	}
	p := &plug{client}
	return p
}

func init() {
	log.SetPrefix("[master log] ")
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func doCall(a, b int, c chan string, plugin *plug) {

	res, err := plugin.Add(a, b)
	if err != nil {
		log.Printf("[FAILURE]: %s", err)
	}
	ret := fmt.Sprintf("[RESULT] %v: %v + %v = %v", a, a, b, res)
	c <- ret
}

func loopStart(ic chan string, client *plug) {
	defer timeTrack(time.Now(), "loopStart")

	for i := 0; i < max; i++ {
		go doCall(i, i+1, ic, client)
	}
}

func main() {
	ic := make(chan string) //a channel that can send and receive an int
	client := createClient()
	defer client.Client.Close()
	loopStart(ic, client)

	var ret string
	for o := 0; o < max; o++ {
		ret = <-ic
		log.Printf("%v", ret)
	}

}

func (p plug) Add(num1, num2 int) (result int, err error) {
	err = p.Client.Call("add", []int{num1, num2}, &result)
	return result, err
}
