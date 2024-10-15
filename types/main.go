package types

type LocateMessage struct {
	Addr string
	Id   int
}

type Bucket struct {
	Key         string
	Doc_count   int
	Min_version struct {
		Value float32
	}
}
