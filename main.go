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

type host struct {
	ip       string
	name     string
	pingInMs int64
}

var hosts = []host{
	{ip: "1.1.1.1", name: "Cloudflare", pingInMs: 9999999},
	{ip: "8.8.8.8", name: "Google", pingInMs: 9999999},
}

func main() {
	c := make(chan []host)

	hostsWithLatency := hosts
	ctx := context.Background()

	ctxWithTimeout, cancelCtx := context.WithTimeout(ctx, 1*time.Second)
	defer cancelCtx()

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	go parallelPing(ctx, conn, c, hosts)

	select {
	case <-ctxWithTimeout.Done():
		hostsWithLatency = hosts
	case hostsWithLatency = <-c:
	}

	latency := int64(0)
	for _, l := range hostsWithLatency {
		latency = latency + l.pingInMs
	}
	print("", latency/int64(len(hostsWithLatency)), false)
	for _, h := range hostsWithLatency {
		print(h.name, h.pingInMs, true)
	}
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
	if text != "" {
		text = text + ": "
	}
	fmt.Printf("%s%dms|color=\"%s\" dropdown=\"%t\" font=\"Hack\"\n", text, ping, color, dropdown)
}

func parallelPing(_ context.Context, conn net.PacketConn, c chan []host, hosts []host) {
	ch := make(chan host, len(hosts))
	for _, host := range hosts {
		go ping(conn, host, 0, ch)
	}

	var pings []host
	for i := 0; i < len(hosts); i++ {
		hostWithPing := <-ch
		pings = append(pings, hostWithPing)
	}

	c <- pings
}

func ping(conn net.PacketConn, h host, seq int, c chan host) {
	defer func() {
		c <- h
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
	if _, err := conn.WriteTo(reqMessageEncoded, &net.UDPAddr{IP: net.ParseIP(h.ip)}); err != nil {
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
	h.pingInMs = resTime.Sub(reqTime).Milliseconds()
}
