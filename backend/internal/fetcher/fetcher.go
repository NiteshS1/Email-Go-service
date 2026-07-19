package fetcher

type Fetcher interface {
	Fetch(url string, name string) (tempPath string, cleanup func(), err error)
}
