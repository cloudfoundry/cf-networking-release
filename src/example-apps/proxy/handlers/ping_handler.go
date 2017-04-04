package handlers

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type PingHandler struct {
}

func ipv4Address(ips []net.IP) (net.Addr, error) {
	for _, ip := range ips {
		if ip.To4() != nil {
			return &net.UDPAddr{IP: ip}, nil
		}
	}
	return nil, errors.New("No IPv4 found")
}

func handleError(err error, destination string, resp http.ResponseWriter) {
	msg := fmt.Sprintf("Ping failed to destination: %s: %s", destination, err)
	fmt.Fprintf(os.Stderr, msg)
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte(msg))
}

func (h *PingHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	destination := strings.TrimPrefix(req.URL.Path, "/ping/")
	destination = strings.Split(destination, ":")[0]

	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		handleError(fmt.Errorf("listen packet: %s", err), destination, resp)
		return
	}
	defer c.Close()

	ips, err := net.LookupIP(destination)
	if err != nil {
		handleError(fmt.Errorf("lookup ip: %s", err), destination, resp)
		return
	}

	ip, err := ipv4Address(ips)
	if err != nil {
		handleError(fmt.Errorf("ipv4 address: %s", err), destination, resp)
		return
	}

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   42,
			Seq:  1,
			Data: []byte("ping-test"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		handleError(fmt.Errorf("marshal icmp message: %s", err), destination, resp)
		return
	}
	if _, err := c.WriteTo(wb, ip); err != nil {
		handleError(fmt.Errorf("write message: %s", err), destination, resp)
		return
	}

	rb := make([]byte, 1500)
	if err := c.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		handleError(fmt.Errorf("set read deadline: %s", err), destination, resp)
		return
	}
	n, peer, err := c.ReadFrom(rb)
	if err != nil {
		handleError(fmt.Errorf("read from: %s", err), destination, resp)
		return
	}
	rm, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		handleError(fmt.Errorf("parse message: %s", err), destination, resp)
		return
	}
	if rm.Type != ipv4.ICMPTypeEchoReply {
		handleError(fmt.Errorf("got %+v from %v; want echo reply", rm, peer), destination, resp)
	}
	resp.Write([]byte(fmt.Sprintf("Ping succeeded to destination: %s", destination)))
}
