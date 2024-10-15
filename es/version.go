package es

import (
	"encoding/json"
	"fmt"
	"go-project/Scalable-distributed-system/types"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type hit struct {
	Source Metadata `json:"_source"`
}

type searchResult struct {
	Hits struct {
		Total int
		Hits  []hit
	}
}

type aggregateResult struct {
	Aggregations struct {
		Group_by_name struct {
			Buckets []types.Bucket
		}
	}
}

func SearchLatestVersion(name string) (meta Metadata, e error) {
	url := fmt.Sprintf("http://%s/metadata/_search?q=Name:%s&size=1&sort=Version:desc",
		os.Getenv("ES_SERVER"), url.PathEscape(name))
	fmt.Println(url)
	r, e := http.Get(url)
	fmt.Println("url:", r)
	if e != nil {
		return
	}
	if r.StatusCode != http.StatusOK {
		e = fmt.Errorf("fail to search latest metadata: %d", r.StatusCode)
		return
	}
	result, _ := ioutil.ReadAll(r.Body)
	var sr searchResult
	json.Unmarshal(result, &sr)
	if len(sr.Hits.Hits) != 0 {
		meta = sr.Hits.Hits[0].Source
	}
	return
}

func AddVersion(name string, hash string, size int64) error {
	version, e := SearchLatestVersion(name)
	if e != nil {
		return e
	}
	return PutMetadata(name, version.Version+1, size, hash)
}
func SearchAllVersions(name string, from, size int) ([]Metadata, error) {
	url := fmt.Sprintf("http://%s/metadata/_search?sort=name,version&from=%d&size=%d", os.Getenv("ES_SERVER"), from, size)
	fmt.Println(url)
	if name != "" {
		url += "&q=Name:" + name
	}
	r, e := http.Get(url)
	if e != nil {
		return nil, e
	}
	metas := make([]Metadata, 0)
	result, _ := ioutil.ReadAll(r.Body)
	var sr searchResult
	json.Unmarshal(result, &sr)
	for i := range sr.Hits.Hits {
		metas = append(metas, sr.Hits.Hits[i].Source)
	}
	fmt.Println(metas)
	return metas, nil
}

func SearchVersionStatus(min_doc_count int) ([]types.Bucket, error) {
	client := http.Client{}
	url := fmt.Sprintf("http://%s/metadata/_search", os.Getenv("ES_SERVER"))
	body := fmt.Sprintf(`
	{
		"size": 0,
		"aggs": {
			"group_by_name": {
				"terms":{
					"field":"name",
					"min_doc_count":%d
				},
				"aggs":{
					"min_version":{
						"min":{
							"filed":"version"
						}
					}
				}
			}	
		}
	}`, min_doc_count)
	request, _ := http.NewRequest("GET", url, strings.NewReader(body))
	r, e := client.Do(request)
	if e != nil {
		return nil, e
	}
	b, _ := ioutil.ReadAll(r.Body)
	var ar aggregateResult
	json.Unmarshal(b, &ar)
	return ar.Aggregations.Group_by_name.Buckets, nil

}
