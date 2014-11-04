package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {

	io.WriteString(w, "Hello world!")
}

func main() {
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./static/"))))
	fmt.Printf("Server listening on http://localhost:8000 \n")

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}

}
