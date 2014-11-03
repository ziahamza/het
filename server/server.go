package main

import (
        "io"
        "net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
    
    io.WriteString(w, "Hello world!")
}

func main() {
    http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./static/"))))
    http.ListenAndServe(":8000", nil)
}
