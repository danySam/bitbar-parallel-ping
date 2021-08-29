package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	c := make(chan int64)
	// ctx := context.Background()

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	go ping(conn, "1.1.1.1", 5233, c)
	go ping(conn, "8.8.8.8", 5894, c)

	t1 := <-c
	t2 := <-c

	fmt.Println("t1", t1)
	fmt.Println("t2", t2)
}

func ping(conn net.PacketConn, ip string, seq int, c chan int64) {

	pingInMs := int64(9999999)
	defer func() {
		c <- pingInMs
	}()

	reqMessage := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   rand.Int() & 0xffff,
			Seq:  seq,
			Data: []byte("Hi"),
		},
	}

	reqMessageEncoded, err := reqMessage.Marshal(nil)
	if err != nil {
		panic(err)
	}

	reqTime := time.Now()
	if _, err := conn.WriteTo(reqMessageEncoded, &net.UDPAddr{IP: net.ParseIP(ip)}); err != nil {
		return
	}

	resTime := time.Now()

	readBuffer := make([]byte, 1500)

	bufferLen, _, err := conn.ReadFrom(readBuffer)
	if err != nil {
		return
	}

	_, err = icmp.ParseMessage(1, readBuffer[:bufferLen])
	if err != nil {
		log.Fatal(err)
	}

	resTime = time.Now()
	pingInMs = resTime.Sub(reqTime).Milliseconds()
}
