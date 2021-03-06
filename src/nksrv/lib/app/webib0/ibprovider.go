package webib0

type IBProvider interface {
	// XXX maybe pass interfaces which are able to accept stuff incrementally instead
	IBGetBoardList(*IBBoardList) (error, int)
	IBGetThreadListPage(*IBThreadListPage, string, uint32) (error, int)
	IBGetOverboardPage(*IBOverboardPage, uint32) (error, int)
	IBGetThreadCatalog(*IBThreadCatalog, string) (error, int)
	IBGetOverboardCatalog(*IBOverboardCatalog) (error, int)
	IBGetThread(*IBThreadPage, string, string) (error, int)
}
