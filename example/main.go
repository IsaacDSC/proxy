package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[*] Request received %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("Request Headers: %+v\n", r.Header)
		body, _ := io.ReadAll(r.Body)
		fmt.Printf("Request Body: %s\n", string(body))
		fmt.Println("--------------------------------")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})
	http.ListenAndServe(":8000", mux)
}
