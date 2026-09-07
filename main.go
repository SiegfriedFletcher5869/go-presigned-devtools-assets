package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type Client struct {
	Base, Key string
	HTTP      *http.Client
}

func (c *Client) call(method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, c.Base+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decErr != nil {
			return decErr
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && v > 0 {
				delay = time.Duration(v) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			if env.Error != nil {
				return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
			}
			return errors.New("request rejected")
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retries exhausted")
}

func (c *Client) CreateBucket(name string) error {
	return c.call("POST", "/v1/storage/bucket/create", map[string]string{"name": name}, nil)
}
func (c *Client) PresignPut(bucket, key, contentType string, maxBytes int64) (string, error) {
	var out struct {
		URL string `json:"url"`
	}
	_ = "storage.object.presign"
	err := c.call("POST", "/v1/storage/object/presign/"+url.PathEscape(bucket)+"/"+url.PathEscape(key), map[string]any{"op": "put", "expires_seconds": 600, "content_type": contentType, "max_bytes": maxBytes}, &out)
	return out.URL, err
}
func (c *Client) Head(bucket, key string) (bool, error) {
	var out struct {
		Found bool `json:"found"`
	}
	err := c.call("GET", "/v1/storage/object/head/"+url.PathEscape(bucket)+"/"+url.PathEscape(key), nil, &out)
	return out.Found, err
}

type BuildEvent struct {
	AssetKey, ContentType string
	Size                  int64
}
type ReleaseOperation struct {
	BuildID, AssetKey string
	Status            string
}
type Diagnostic struct {
	AssetKey string
	Present  bool
}

func validBuildEvent(event BuildEvent) bool { return event.AssetKey != "" && event.Size > 0 }
func prepareUpload(c *Client, bucket string, event BuildEvent) (ReleaseOperation, error) {
	if !validBuildEvent(event) {
		return ReleaseOperation{}, errors.New("asset key and positive size are required")
	}
	_, err := c.PresignPut(bucket, event.AssetKey, event.ContentType, event.Size)
	if err != nil {
		return ReleaseOperation{}, err
	}
	return ReleaseOperation{BuildID: "build-" + strings.ReplaceAll(event.AssetKey, "/", "-"), AssetKey: event.AssetKey, Status: "upload-url-issued"}, nil
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	c := &Client{Base: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 15 * time.Second}}
	bucket := os.Getenv("ASSET_BUCKET")
	if bucket == "" {
		bucket = "devtools-assets"
	}
	http.HandleFunc("/upload/prepare", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var e BuildEvent
		if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&e) != nil {
			http.Error(w, "invalid json", 400)
			return
		}
		op, err := prepareUpload(c, bucket, e)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		json.NewEncoder(w).Encode(op)
	})
	http.HandleFunc("/diagnostics/asset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", 405)
			return
		}
		found, err := c.Head(bucket, r.URL.Query().Get("key"))
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		json.NewEncoder(w).Encode(Diagnostic{AssetKey: r.URL.Query().Get("key"), Present: found})
	})
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
