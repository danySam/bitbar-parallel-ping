package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	c := make(chan []int64)

	latencies := []int64{9999999, 9999999}
	ctx := context.Background()

	ctxWithTimeout, cancelCtx := context.WithTimeout(ctx, 1*time.Second)
	defer cancelCtx()

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	go pingWithTimeout(ctx, conn, c)

	select {
	case <-ctxWithTimeout.Done():
		latencies = []int64{10000, 10000}
	case latencies = <-c:
	}

	latency := int64(0)
	for _, l := range latencies {
		latency = latency + l
	}
	print("", latency/int64(len(latencies)), false)
	print("Google: ", latencies[0], true)
	print("Cloudflare: ", latencies[1], true)
}

func print(text string, ping int64, dropdown bool) {
	color := "green"
	switch {
	case ping < 100:
		color = "green"
	case ping < 500:
		color = "orange"
	case ping < 1000:
		color = "purple"
	default:
		color = "red"
	}
	fmt.Printf("%s%dms|color=\"%s\" dropdown=\"%t\" font=\"Hack\"\n", text, ping, color, dropdown)
}

func pingWithTimeout(_ context.Context, conn net.PacketConn, c chan []int64) {
	ch1 := make(chan int64)
	ch2 := make(chan int64)
	go ping(conn, "8.8.8.8", 5894, ch1)
	go ping(conn, "1.1.1.1", 5233, ch2)

	t1, t2 := <-ch1, <-ch2
	c <- []int64{t1, t2}
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
