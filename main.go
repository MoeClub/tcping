package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

type Summary struct {
	NET      string
	MAX      time.Duration
	MIN      time.Duration
	AVG      time.Duration
	SUM      time.Duration
	Count    int
	ErrCount int
	Time     time.Time
	Lock     *sync.Mutex
	WG       *sync.WaitGroup
}

func (p *Ping) Resolver() error {
	var r *net.Resolver
	var c bool
	if DefaultDNSAddr != "" && DefaultDNSNet != "" {
		dialer := &net.Dialer{}
		r = &net.Resolver{
			PreferGo:     true,
			StrictErrors: false,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.DialContext(ctx, strings.ToLower(DefaultDNSNet), DefaultDNSAddr)
			},
		}
		c = true
	} else {
		r = &net.Resolver{}
	}
	t := time.Now()
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
			if c {
				fmt.Printf("[DNS] [%s] %s --> %s - %s\n", DefaultDNSAddr, p.host, p.addr, time.Since(t))
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
	dialer := &net.Dialer{
		Timeout: time.Duration(p.timeout) * time.Second,
	}
	stats.Host = p.host
	stats.DAddr = fmt.Sprintf("%s:%d", p.addr, p.port)
	stats.Time = time.Now()
	conn, err := dialer.DialContext(context.Background(), p.net, stats.DAddr)
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

func (s *Summary) Stats() {
	count := s.Count - s.ErrCount
	if count > 0 {
		s.AVG = s.SUM / time.Duration(count)
	}
	fmt.Printf("\n[%s] Max: %s Min: %s Avg: %s Total: %d Error: %d - %s\n\n", strings.ToUpper(s.NET), s.MAX, s.MIN, s.AVG, s.Count, s.ErrCount, time.Since(s.Time))
}

func (s *Summary) Result(ping *Ping) {
	defer s.WG.Done()
	stats := ping.Ping()
	s.Lock.Lock()
	s.Count += 1
	if stats.Error == nil {
		fmt.Printf("[%s] %s --> %s - %s\n", stats.Time.Format("2006/01/02 15:04:05"), stats.SAddr, stats.DAddr, stats.Duration)
		if s.MIN > stats.Duration || s.MIN == 0 {
			s.MIN = stats.Duration
		}
		if s.MAX < stats.Duration {
			s.MAX = stats.Duration
		}

		s.SUM += stats.Duration
	} else {
		s.ErrCount += 1
		fmt.Printf("[%s] %s:%d - %s\n", stats.Time.Format("2006/01/02 15:04:05"), ping.host, ping.port, stats.Error.Error())
	}
	s.Lock.Unlock()
	return
}

func (p *Ping) Do(s *Summary) {
	err := p.Resolver()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer s.Stats()
	defer s.WG.Wait()
	i := 0
	for {
		s.WG.Add(1)
		if DefaultInterval > 0 {
			s.Result(p)
			time.Sleep(time.Duration(DefaultInterval) * time.Second)
		} else {
			go s.Result(p)
		}
		if DefaultCount >= 0 {
			i += 1
			if DefaultCount <= i {
				break
			}
		}
	}
	return
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

	if DefaultHost == "" {
		switch flag.NArg() {
		case 1:
			DefaultHost = flag.Args()[0]
		case 2:
			DefaultHost = flag.Args()[0]
			prot, err := strconv.Atoi(flag.Args()[1])
			if err != nil {
				DefaultHost = ""
			}
			DefaultPort = prot
		default:
			DefaultHost = ""
		}
	}

	if DefaultHost == "" || DefaultPort == 0 {
		fmt.Printf("Use '-h' to set host, '-p' to set port.\n")
		os.Exit(127)
	}
}

func (s *Summary) Interrupt() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		s.Stats()
		os.Exit(1)
	}()
}

func main() {
	ping := Ping{
		net:     strings.ToLower(DefaultNet),
		host:    DefaultHost,
		port:    DefaultPort,
		timeout: DefaultTimeout,
	}
	summary := &Summary{
		NET:  ping.net,
		Time: time.Now(),
		Lock: &sync.Mutex{},
		WG:   &sync.WaitGroup{},
	}
	summary.Interrupt()
	ping.Do(summary)
}
