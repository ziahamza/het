package het

type CountStats struct {
	DocumentCount, PendingCount, KeywordCount int
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
	Title        string
	ModifiedDate string
	Size         int
	Keywords     []KeywordRef
}

// stored in keywords bucket
type Keyword struct {
	Frequency int
	Docs      []DocumentRef
}
