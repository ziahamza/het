package main

import (
        "fmt"
        "log"
        "net/http"
        "net/url"
        "strings"
        
        "code.google.com/p/go.net/html"
        "github.com/boltdb/bolt"
        )

func main() {
    db, err := bolt.Open("./index.db", 0600, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    defer db.Close()
    
    err = db.Update(func(tx *bolt.Tx) error {
        fmt.Printf("creating db ... \n")
        docs, err := tx.Bucket([]byte("docs"))
        if err != nil {
            return err
        }
        
        resultFile, _ = File.Open("./spider_result.txt", O_WRONLY, 0666);
        
        docs.forEach
        
        return nil
    })
    
}
