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

// made a new type to sort out the data
type KeywordList []KeywordRef

func (a KeywordList) Len() int           { return len(a) }
func (a KeywordList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a KeywordList) Less(i, j int) bool { return a[i].Frequency >= a[j].Frequency }

// stored in docs bucket
type Document struct {
	Title        string
	LastModified string
	Size         int
	Keywords     KeywordList
	ChildLinks   []string
}

// stored in keywords bucket
type Keyword struct {
	Frequency int
	Docs      []DocumentRef
}
