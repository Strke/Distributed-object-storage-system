package rs

import (
	"fmt"
	"github.com/klauspost/reedsolomon"
	"go-project/Scalable-distributed-system/objectstream"
	"io"
)

type RSGetStream struct {
	*decoder
}

func (s *RSGetStream) Close() {
	for i := range s.writers {
		if s.writers[i] != nil {
			s.writers[i].(*objectstream.TempPutStream).Commit(true)
		}
	}
}

type decoder struct {
	readers   []io.Reader
	writers   []io.Writer
	enc       reedsolomon.Encoder //接口，用于RS解码
	size      int64               //对象的大小
	cache     []byte              //缓存的数据
	cacheSize int                 //缓存数据的大小
	total     int64               //表示已经读取了多少字节
}

//locateInfo数组中表示含有指定对象的数据服务节点，
//dataServers表示选定的用来接收恢复分片的服务器节点

func NewRSGetStream(locateInfo map[int]string, dataServers []string, hash string, size int64) (*RSGetStream, error) {
	if len(locateInfo)+len(dataServers) != ALL_SHARDS {
		return nil, fmt.Errorf("dataServers number mismatch")
	}
	readers := make([]io.Reader, ALL_SHARDS)
	for i := 0; i < ALL_SHARDS; i++ {
		server := locateInfo[i]
		//将locateInfo和dataServers统合到locateInfo中
		if server == "" {
			locateInfo[i] = dataServers[0]
			dataServers = dataServers[1:]
			continue
		}
		//从这些数据服务节点中获取对象内容，并保存在readers数组上
		reader, e := objectstream.NewGetStream(server, fmt.Sprintf("%s.%d", hash, i))
		if e == nil {
			readers[i] = reader
		}
	}
	writers := make([]io.Writer, ALL_SHARDS)
	perShard := (size + DATA_SHARDS - 1) / DATA_SHARDS
	var e error
	for i := range readers {
		//如果某个数据服务节点不可读，则为其开启写通道
		if readers[i] == nil {
			writers[i], e = objectstream.NewTempPutStream(locateInfo[i], fmt.Sprintf("%s.%d", hash, i), perShard)
			if e != nil {
				return nil, e
			}
		}
	}
	//至此，程序为能够读取内容的数据服务节点开启了读通道，为不能读取内容的服务节点开启了写通道

	dec := NewDecoder(readers, writers, size)
	return &RSGetStream{dec}, nil
}

func NewDecoder(readers []io.Reader, writers []io.Writer, size int64) *decoder {
	enc, _ := reedsolomon.New(DATA_SHARDS, PARITY_SHARDS)
	return &decoder{readers, writers, enc, size, nil, 0, 0}
}

func (d *decoder) Read(p []byte) (n int, err error) {
	if d.cacheSize == 0 {
		e := d.getData()
		if e != nil {
			return 0, e
		}
	}
	length := len(p)
	if d.cacheSize < length {
		length = d.cacheSize
	}
	d.cacheSize -= length
	copy(p, d.cache[:length])
	d.cache = d.cache[:length]
	return length, nil
}

func (d *decoder) getData() error {
	if d.total == d.size {
		return io.EOF
	}
	shards := make([][]byte, ALL_SHARDS)
	repairIds := make([]int, 0)
	for i := range shards {
		if d.readers[i] == nil {
			repairIds = append(repairIds, i)
		} else {
			shards[i] = make([]byte, BLOCK_PER_SHARD)
			n, e := io.ReadFull(d.readers[i], shards[i])
			if e != nil && e != io.EOF && e != io.ErrUnexpectedEOF {
				shards[i] = nil
			} else if n != BLOCK_PER_SHARD {
				shards[i] = shards[i][:n]
			}
		}
	}
	//Reconstruct函数可以尝试修复数据服务节点上被破坏的内容
	e := d.enc.Reconstruct(shards)
	//未修复成功的话就报告
	if e != nil {
		fmt.Println("data can`t be repaired")
		return e
	}
	fmt.Println("required repair id:", repairIds)
	for i := range repairIds {
		id := repairIds[i]
		fmt.Println(string(shards[id]))
		d.writers[id].Write(shards[id])
	}
	for i := 0; i < DATA_SHARDS; i++ {
		shardSize := int64(len(shards[i]))
		if d.total+shardSize > d.size {
			shardSize -= d.total + shardSize - d.size
		}
		d.cache = append(d.cache, shards[i][:shardSize]...)
		d.cacheSize += int(shardSize)
		d.total += shardSize
	}
	return nil
}

func (s *RSGetStream) Seek(offset int64, whence int) (int64, error) {
	if whence != io.SeekCurrent {
		panic("only support SeekCurrent")
	}
	if offset < 0 {
		panic("only support forward seek")
	}
	for offset != 0 {
		length := int64(BLOCK_SIZE)
		if offset < length {
			length = offset
		}
		buf := make([]byte, length)
		io.ReadFull(s, buf)
		offset -= length
	}
	return offset, nil
}
