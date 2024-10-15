package rs

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/klauspost/reedsolomon"
	"go-project/Scalable-distributed-system/objectstream"
	"io"
)

//内嵌encoder指针，相当于继承，
//RSPutStream的使用者也能访问encoder的方法和成员

type RSPutStream struct {
	*encoder
}

type encoder struct {
	writers []io.Writer
	enc     reedsolomon.Encoder
	cache   []byte
}

func NewRSPutStream(dataServers []string, hash string, size int64) (*RSPutStream, error) {
	if len(dataServers) != ALL_SHARDS {
		return nil, fmt.Errorf("dataServers number mismatch")
	}
	perShard := (size + DATA_SHARDS - 1) / DATA_SHARDS
	writers := make([]io.Writer, ALL_SHARDS)
	var e error
	for i := range writers {
		//实现了Write方法的对象都可以存放在io.Writer类型中,此处的writers[i]是一个TempPutStream对象
		//每一个writers[i]，都用于给数据服务节点上传一个分片
		writers[i], e = objectstream.NewTempPutStream(dataServers[i], fmt.Sprintf("%s.%d", hash, i), perShard)
		if e != nil {
			return nil, e
		}
	}
	enc := NewEncoder(writers)
	return &RSPutStream{enc}, nil
}

func NewEncoder(writers []io.Writer) *encoder {
	enc, _ := reedsolomon.New(DATA_SHARDS, PARITY_SHARDS)
	return &encoder{writers, enc, nil}
}

func (e *encoder) Write(p []byte) (n int, err error) {
	length := len(p)
	current := 0
	for length != 0 {
		next := BLOCK_SIZE - len(e.cache)
		if next > length {
			next = length
		}
		e.cache = append(e.cache, p[current:current+next]...)
		if len(e.cache) == BLOCK_SIZE {
			e.Flush()
		}
		current += next
		length -= next
	}
	return len(p), nil
}

func (e *encoder) Flush() {
	if len(e.cache) == 0 {
		return
	}
	h := sha256.New()
	h.Write(e.cache)
	fmt.Println("the hash of data has not storage", base64.StdEncoding.EncodeToString(h.Sum(nil)))
	shards, _ := e.enc.Split(e.cache)
	e.enc.Encode(shards)
	for i := range shards {
		e.writers[i].Write(shards[i])
	}
	e.cache = []byte{}
}

func (s *RSPutStream) Commit(success bool) {
	s.Flush()
	for i := range s.writers {
		//类型断言，由于TempPutStream类型的变量实现了Write方法，所以我们在前面可以把他存在io.writer中
		//所以这里使用类型断言必定不会报错，并且把io.writer变量转换为TempPutStream变量，
		//并进一步的调用了TempPutStream的Commit方法
		s.writers[i].(*objectstream.TempPutStream).Commit(success)
	}
}

func NewRSResumablePutStreamFromToken(token string) (*RSResumablePutStream, error) {
	b, e := base64.StdEncoding.DecodeString(token)
	if e != nil {
		fmt.Println("decode error")
		return nil, e
	}
	var t resumableToken
	e = json.Unmarshal(b, &t)
	fmt.Println("the token send to put means:", t)
	if e != nil {
		fmt.Println("json parse error")
		return nil, e
	}
	writers := make([]io.Writer, ALL_SHARDS)
	for i := range writers {
		writers[i] = &objectstream.TempPutStream{t.Servers[i], t.Uuids[i]}
	}
	enc := NewEncoder(writers)
	return &RSResumablePutStream{&RSPutStream{enc}, &t}, nil
}
