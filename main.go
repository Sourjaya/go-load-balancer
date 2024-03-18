package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(w http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func newSimpleServer(addr string) *simpleServer {
	serverURL, err := url.Parse(addr)
	handleErr(err)
	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (s *simpleServer) Address() string { return s.addr }
func (s *simpleServer) IsAlive() bool {
	response, err := http.Get(s.addr)
	if err != nil {
		log.Println("Error: Checking url")
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == 200
}
func (s *simpleServer) Serve(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		log.Println(lb.roundRobinCount)
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	log.Println(lb.roundRobinCount)
	lb.roundRobinCount++
	return server
}
func (lb *LoadBalancer) serveProxy(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextAvailableServer()
	log.Printf("forwarding request to address %q\n", target.Address())
	target.Serve(w, r)
}

func handleErr(err error) {
	if err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.youtube.com"),
		newSimpleServer("https://www.google.co.in"),
	}
	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		lb.serveProxy(w, r)
	}
	http.HandleFunc("/", handleRedirect)
	log.Printf("serving request at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
