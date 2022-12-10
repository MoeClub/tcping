package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var (
	DefaultTimeout  = 3
	DefaultHost     = ""
	DefaultPort     = 80
	DefaultNet      = "tcp"
	DefaultInterval = 1
	DefaultCount    = 10
	DefaultDNSNet   = "udp"
	DefaultDNSAddr  = ""
)

type Ping struct {
	net     string
	host    string
	addr    string
	port    int
	dialer  *net.Dialer
	timeout int
}

type Stats struct {
	Time     time.Time
	Duration time.Duration
	Host     string
	SAddr    string
	DAddr    string
	Error    error
}

func (p *Ping) Resolver() error {
	var r *net.Resolver
	if DefaultDNSAddr != "" && DefaultDNSNet != "" {
		dialer := &net.Dialer{}
		r = &net.Resolver{
			PreferGo:     true,
			StrictErrors: false,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.DialContext(ctx, DefaultDNSNet, DefaultDNSAddr)
			},
		}
	} else {
		r = &net.Resolver{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(DefaultTimeout)*time.Second)
	defer cancel()
	addr, err := r.LookupHost(ctx, p.host)
	if err == nil {
		if len(addr) > 0 {
			if strings.ContainsRune(addr[0], ':') {
				p.addr = fmt.Sprintf("[%s]", addr[0])
			} else {
				p.addr = addr[0]
			}
			return err
		}
		err = errors.New("not found addr")
	}
	return err
}

func (p *Ping) Ping() *Stats {
	stats := &Stats{
		Time: time.Now(),
	}
	if p.addr == "" {
		stats.Error = errors.New("invalid host")
		return stats
	}
	if p.port <= 0 {
		p.port = DefaultPort
	}
	if p.timeout <= 0 {
		p.timeout = DefaultTimeout
	}
	if p.net == "" {
		p.net = DefaultNet
	}
	p.dialer = &net.Dialer{
		Timeout: time.Duration(p.timeout) * time.Second,
	}
	stats.Host = p.host
	stats.DAddr = fmt.Sprintf("%s:%d", p.addr, p.port)
	stats.Time = time.Now()
	conn, err := p.dialer.DialContext(context.Background(), p.net, stats.DAddr)
	stats.Duration = time.Since(stats.Time)
	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		stats.Error = err
	} else {
		stats.SAddr = conn.LocalAddr().String()
	}
	return stats
}

func (p *Ping) Do() {
	err := p.Resolver()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	i := 0
	for {
		stats := p.Ping()
		if stats.Error == nil {
			fmt.Printf("[%s] [%s] %s --> %s - %s\n", stats.Time.Format("2006/01/02 15:04:05.999"), strings.ToUpper(p.net), stats.SAddr, stats.DAddr, stats.Duration)
		} else {
			fmt.Printf("[%s] [%s] %s:%d - %s\n", stats.Time.Format("2006/01/02 15:04:05.999"), strings.ToUpper(p.net), p.host, p.port, stats.Error.Error())
		}
		if DefaultCount > 0 {
			i += 1
			if DefaultCount <= i {
				break
			}
		}
		if DefaultInterval > 0 {
			time.Sleep(time.Duration(DefaultInterval) * time.Second)
		}
	}
}

func init() {
	flag.StringVar(&DefaultDNSAddr, "dns", "", "Use DNS IP:PORT")
	flag.StringVar(&DefaultDNSNet, "dns-net", "udp", "Use DNS Net")
	flag.StringVar(&DefaultNet, "n", "tcp", "Use Net")
	flag.StringVar(&DefaultHost, "h", "", "Ping Host")
	flag.IntVar(&DefaultInterval, "i", 1, "Ping Interval")
	flag.IntVar(&DefaultTimeout, "w", 1, "Ping Timeout")
	flag.IntVar(&DefaultCount, "c", 10, "Ping Count")
	flag.IntVar(&DefaultPort, "p", 80, "Default TCP Port.")
	flag.Parse()

	if DefaultHost == "" && flag.NArg() == 1 {
		DefaultHost = flag.Args()[0]
	}

	if DefaultHost == "" {
		fmt.Printf("Use '-h' to set host, '-p' to set port.\n")
		os.Exit(1)
	}
}

func main() {
	ping := Ping{
		net:     DefaultNet,
		host:    DefaultHost,
		port:    DefaultPort,
		timeout: DefaultTimeout,
	}
	ping.Do()
}