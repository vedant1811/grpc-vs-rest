package main

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/Bimde/grpc-vs-rest/pb"
	"golang.org/x/net/http2"
)

var client http.Client

func init() {
	client = http.Client{}
}

// This code was taken from https://posener.github.io/http2/
func createTLSConfigWithCustomCert() *tls.Config {
	// Create a pool with the server certificate since it is not signed
	// by a known CA
	caCert, err := ioutil.ReadFile("server/server.crt")
	if err != nil {
		log.Fatalf("Reading server certificate: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration with the certificate of the server
	return &tls.Config{
		RootCAs: caCertPool,
	}
}

// func BenchmarkHTTP2Get(b *testing.B) {
// 	client.Transport = &http2.Transport{
// 		TLSClientConfig: createTLSConfigWithCustomCert(),
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(b.N)
// 	for i := 0; i < b.N; i++ {
// 		go func() {
// 			get("https://localhost:8080", &pb.Random{})
// 			wg.Done()
// 		}()
// 	}
// 	wg.Wait()
// }

func get(path string, output interface{}) error {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		log.Println("error creating request ", err)
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println("error executing request ", err)
		return err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading response body ", err)
		return err
	}

	err = json.Unmarshal(bytes, output)
	if err != nil {
		log.Println("error unmarshalling response ", err)
		return err
	}

	return nil
}

type Request struct {
	Path   string
	Random *pb.Random
}

const stopRequestPath = "STOP"
const noWorkers = 4096

func BenchmarkHTTP2GetWithWokers(b *testing.B) {
	client.Transport = &http2.Transport{
		TLSClientConfig: createTLSConfigWithCustomCert(),
	}
	requestQueue := make(chan Request)
	defer startWorkers(&requestQueue, noWorkers, startWorker)()
	b.ResetTimer() // don't count worker initialization time
	for i := 0; i < b.N; i++ {
		requestQueue <- Request{Path: "https://localhost:8080", Random: &pb.Random{}}
	}
}

func BenchmarkHTTP11Get(b *testing.B) {
	client.Transport = &http.Transport{
		TLSClientConfig: createTLSConfigWithCustomCert(),
	}
	requestQueue := make(chan Request)
	defer startWorkers(&requestQueue, noWorkers, startWorker)()
	b.ResetTimer() // don't count worker initialization time
	for i := 0; i < b.N; i++ {
		requestQueue <- Request{Path: "https://localhost:8080", Random: &pb.Random{}}
	}
}

func startWorkers(requestQueue *chan Request, noWorkers int, startWorker func(*chan Request, *sync.WaitGroup)) func() {
	var wg sync.WaitGroup
	for i := 0; i < noWorkers; i++ {
		startWorker(requestQueue, &wg)
	}
	return func() {
		wg.Add(noWorkers)
		stopRequest := Request{Path: stopRequestPath}
		for i := 0; i < noWorkers; i++ {
			*requestQueue <- stopRequest
		}
		wg.Wait()
	}
}

func startWorker(requestQueue *chan Request, wg *sync.WaitGroup) {
	go func() {
		for {
			request := <-*requestQueue
			if request.Path == stopRequestPath {
				wg.Done()
				return
			}
			get(request.Path, request.Random)
		}
	}()
}
