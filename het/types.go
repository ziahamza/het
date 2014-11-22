package het

import "net/url"

type CountStats struct {
	DocumentCount, PendingCount, KeywordCount int
}

// used by the docs bucket to refer to a specific keyword under a document
type KeywordRef struct {
	Word      string
	Frequency int
}

// stored in docs bucket
type Document struct {
	URL    url.URL
	Title  string
	Length float64 // normalized length based on word vector
	Size   int
}

// stored in the links bucket.
type Link struct {
	// used if this a redirect link. If it is than the rest of the fields will be empty ...
	ContentType  string
	LastModified string
	StatusCode   int
	URL          url.URL

	// if true, then the redirected URL contains the redirected link. The rest of fields are false
	Redirect bool

	Rank float64

	Outgoing  map[string]bool
	Incomming map[string]bool
}

// stored in bucket doc-keywords mapped by the doc url
type DocKeywords []KeywordRef

func (a DocKeywords) Len() int           { return len(a) }
func (a DocKeywords) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DocKeywords) Less(i, j int) bool { return a[i].Frequency >= a[j].Frequency }

// used by the keywords bucket to refer to a document containing a specific keyword
type DocumentRef struct {
	URL       url.URL
	Frequency int
}

// stored in keywords bucket
type Keyword struct {
	Frequency int
	Docs      []DocumentRef
}
