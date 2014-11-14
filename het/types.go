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
	LastModified string
	Size         int
}

// stored in buucket doc-links
type DocLinks []string

// stored in bucket doc-keywords mapped by the doc url
type DocKeywords []KeywordRef

func (a DocKeywords) Len() int           { return len(a) }
func (a DocKeywords) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DocKeywords) Less(i, j int) bool { return a[i].Frequency >= a[j].Frequency }

// stored in keywords bucket
type Keyword struct {
	Frequency int
	Docs      []DocumentRef
}
