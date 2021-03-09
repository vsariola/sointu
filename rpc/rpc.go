package rpc

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type SyncServer struct {
	channel chan []float32
}

func (s *SyncServer) Sync(syncData []float32, reply *int) error {
	select {
	case s.channel <- syncData:
	default:
	}
	return nil
}

func Receiver() (<-chan []float32, error) {
	c := make(chan []float32, 1)
	server := &SyncServer{channel: c}
	rpc.Register(server)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":31337")
	if e != nil {
		log.Fatal("listen error:", e)
		return nil, fmt.Errorf("net.listen failed: %v", e)
	}
	go func() {
		defer close(c)
		http.Serve(l, nil)
	}()
	return c, nil
}

func Sender(serverAddress string) (chan<- []float32, error) {
	c := make(chan []float32, 256)
	client, err := rpc.DialHTTP("tcp", serverAddress+":31337")
	if err != nil {
		log.Fatal("dialing:", err)
		return nil, fmt.Errorf("rpc.DialHTTP failed: %v", err)
	}
	go func() {
		for msg := range c {
			var reply int
			err = client.Call("SyncServer.Sync", msg, &reply)
			if err != nil {
				log.Fatal("SyncServer.Sync error:", err)
			}
		}
	}()
	return c, nil
}
