package main

import (
        "encoding/json"
        "fmt"
        "log"
        "os"
        
        "github.com/boltdb/bolt"
        )


type Stats struct {
    DocumentCount, KeywordCount int
}

// used by the docs bucket to refer to a specific keyword under a document
type KeywordRef struct {
    Word      string
    Frequency int
}

// used by the keywords bucket to refer to a document containing a specific keyword
type DocumentRef struct {
    URL       string
    Frequency int
}

// stored in docs bucket
type Document struct {
    Title    string
    Size     int
    Keywords []KeywordRef
}

// stored in keywords bucket
type Keyword struct {
    Frequency int
    Docs      []DocumentRef
}


func main() {
    db, err := bolt.Open("./index.db", 0600, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    defer db.Close()
    
    err = db.Update(func(tx *bolt.Tx) error {
        fmt.Printf("creating db ... \n")
        docs := tx.Bucket([]byte("docs"))
        
        resultFile, err := os.OpenFile("./spider_result.txt", os.O_WRONLY|os.O_CREATE, 0666);
        if err != nil {
            log.Fatal(err);
            return nil
        }
        
        docs.ForEach(func(k, v []byte) error {
            doc := Document{}
            json.Unmarshal(v, &doc)
            
            resultFile.WriteString(doc.Title+"\n")
            resultFile.WriteString(string(k)+"\n")
            
            for _, kw :=range doc.Keywords {
                resultFile.WriteString(kw.Word+" "+fmt.Sprintf("%d", kw.Frequency)+";")
            }
            resultFile.WriteString("\n")
            
            resultFile.WriteString("-------------------------------------------------------------------------------------------\n")
            
            return nil
        })
        
        resultFile.Close()
        
        return nil
    })
    
}
