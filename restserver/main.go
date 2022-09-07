package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Bimde/grpc-vs-rest/pb"
)

func handle(w http.ResponseWriter, req *http.Request) {
	log.Print("handle")
	decoder := json.NewDecoder(req.Body)
	var random pb.Random
	if err := decoder.Decode(&random); err != nil {
		panic(err)
	}
	random.RandomString = "[Updated] " + random.RandomString

	bytes, err := json.Marshal(&random)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func main() {
	server := &http.Server{Addr: "localhost:8080", Handler: http.HandlerFunc(handle)}
	log.Print("main")
	log.Fatal(server.ListenAndServeTLS("../server/server.crt", "../server/server.key"))
}
