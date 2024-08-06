package oss

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestUpload(t *testing.T) {
	endpoint := os.Getenv("endpoint")
	region := os.Getenv("region")
	accessKey := os.Getenv("accessKey")
	secretKey := os.Getenv("secretKey")
	bucket := os.Getenv("bucket")
	domain := os.Getenv("domain")
	// fail if endpoint/region/accessKey/secretKey/bucket is empty
	if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Errorf("endpoint/region/accessKey/secretKey/bucket/domain is empty")
		t.Failed()
		return
	}
	// output accessKey/secretKey/bucket
	t.Logf("endpoint: %s, region: %s, accessKey: %s, secretKey: %s, bucket: %s, domain: %s", endpoint, region, accessKey, secretKey, bucket, domain)
	// upload a file
	content := fmt.Sprintf("hello: %d", time.Now().UnixNano())
	fileData := []byte(content)
	fileKey := "hello.txt"

	client := &Oss{Endpoint: endpoint, Region: region, AccessKey: accessKey, SecretKey: secretKey,
		Bucket: bucket, Domain: domain}
	url, err := client.Upload(fileData, fileKey)
	if err != nil {
		t.Errorf("upload failed: %s", err)
		t.Failed()
		return
	}
	// output url
	t.Logf("url: %s", url)
	// get file content from url
	resp, err := http.Get(url + "?id=" + fmt.Sprintf("%d", time.Now().UnixNano()))
	if err != nil {
		t.Errorf("http.Get failed: %s", err)
		t.Failed()
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("io.ReadAll failed: %s", err)
		t.Failed()
		return
	}
	// 打印文件内容
	t.Logf("文件内容: %s", string(body))
	if string(body) != content {
		t.Errorf("file content not match")
		t.Failed()
		return
	}

}
