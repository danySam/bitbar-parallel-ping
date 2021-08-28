package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	c := make(chan int64)
	ctx := context.Background()
	go pingWithTimeout(ctx, "1.1.1.1", 53, c)
	go pingWithTimeout(ctx, "8.8.8.8", 589084, c)

	t1, t2 := <-c, <-c
	fmt.Println("t1", t1)
	fmt.Println("t2", t2)
}

func pingWithTimeout(ctxMain context.Context, ip string, seq int, c chan int64) int64 {
	ctx, cancel := context.WithTimeout(ctxMain, 1*time.Second)
	defer cancel()
	c2 := make(chan int64)
	go ping(ctx, ip, seq, c2)

	select {
	case <-ctx.Done():
		c <- 9999999
		return 9999999

	case t := <-c2:
		c <- t
		return t
	}
}
func ping(_ context.Context, ip string, seq int, c chan int64) int64 {

	pingInMs := int64(9999999)
	defer func() {
		fmt.Println(pingInMs)
		c <- pingInMs
	}()

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	reqTime := time.Now()

	reqMessage := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid(),
			Seq:  seq,
			Data: []byte("Hi"),
		},
	}
	reqMessageEncoded, err := reqMessage.Marshal(nil)
	if err != nil {
		panic(err)
	}

	if _, err := conn.WriteTo(reqMessageEncoded, &net.UDPAddr{IP: net.ParseIP(ip)}); err != nil {
		return pingInMs
	}

	readBuffer := make([]byte, 1500)
	bufferLen, peer, err := conn.ReadFrom(readBuffer)
	if err != nil {
		return pingInMs
	}

	resMessage, err := icmp.ParseMessage(1, readBuffer[:bufferLen])
	if err != nil {
		log.Fatal(err)
	}

	resTime := time.Now()

	// 	fmt.Println("peer", peer)
	// 	fmt.Println("resType", resMessage.Type)
	fmt.Println("-------------------")
	fmt.Println(peer, bufferLen)
	fmt.Printf("Pointer: %p Pointer for peer: %p P for conn: %p\n", &readBuffer, &peer, &conn)
	fmt.Println(resMessage.Body.Marshal(-1))
	fmt.Println(resTime)
	fmt.Println("-------------------")
	body, _ := resMessage.Body.Marshal(20)
	fmt.Println("body", body)
	fmt.Printf("got %+v; want echo reply", readBuffer[:bufferLen])

	fmt.Println(conn)
	pingInMs = resTime.Sub(reqTime).Milliseconds()
	fmt.Println("pingBefore", pingInMs)
	return pingInMs
}
