package connector

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"strings"
)

type S3Connection struct {
	Connection     *s3.S3
	CacheDirectory string
	Downloader     *s3manager.Downloader
}

func NewS3Connection(connection *s3.S3, cacheDirectory string,
	options *s3manager.DownloadOptions) *S3Connection {
	conn := S3Connection{
		Connection:     connection,
		CacheDirectory: cacheDirectory,
	}
	conn.Downloader = s3manager.NewDownloader(options)
	return &conn
}

type S3Object struct {
	S3FullPath string
}

func (s3Obj S3Object) String() string {
	return fmt.Sprintf("{S3Object: S3Object=%s}", s3Obj.S3FullPath)
}

func (conn *S3Connection) ListObjectsAsFolders(bucket string, prefix string) ([]S3Object, error) {
	return conn.ListObjects(bucket, prefix, "/")
}

func (conn *S3Connection) ListObjectsAsAll(bucket string, prefix string) ([]S3Object, error) {
	return conn.ListObjects(bucket, prefix, ",")
}

func (conn *S3Connection) ListObjects(bucket string, prefix string, delimiter string) ([]S3Object, error) {
	s3Objs := make([]S3Object, 0)
	moreResults := false
	nextMarker := aws.String("")
	for {
		input := s3.ListObjectsInput{
			Bucket:    aws.String(bucket),
			Prefix:    aws.String(prefix),
			Delimiter: aws.String(delimiter),
		}
		if moreResults {
			input.Marker = nextMarker
		}
		result, err := conn.Connection.ListObjects(&input)
		if err != nil {
			log.Debugln("Failed to ListObjects for bucket %s, prefix %s: %s", bucket, prefix, err)
			return nil, err
		}
		if delimiter == "/" {  // folders
			for _, commonPrefix := range result.CommonPrefixes {
				s3Obj := S3Object{
					S3FullPath: strings.TrimSuffix(*commonPrefix.Prefix, "/"),
				}
				s3Objs = append(s3Objs, s3Obj)
			}
		} else { // regular files
			for _, contents := range result.Contents {
				s3Obj := S3Object{
					S3FullPath: *contents.Key,
				}
				s3Objs = append(s3Objs, s3Obj)
			}
		}
		time.Sleep(100 * time.Millisecond)
		moreResults = *result.IsTruncated
		if moreResults {
			nextMarker = result.NextMarker
		} else {
			break
		}
	}
	return s3Objs, nil
}

func (conn *S3Connection) getCacheFilepath(key string) (string, error) {
	cacheFilepath := conn.CacheDirectory + "/" + key
	cacheFilepath, err := filepath.Abs(cacheFilepath)
	if err != nil {
		log.Debugf("Failed to make cacheFilepath %s absolute: %s",
			cacheFilepath, err)
		return "", err
	}
	return cacheFilepath, nil
}

func (conn *S3Connection) CachedGet(bucket string, key string) (string, error) {
	cacheFilepath, err := conn.getCacheFilepath(key)
	if err != nil {
		log.Debugf("Failed to getCacheFilepath in CachedGet: %s", err)
		return "", err
	}
	if _, err := os.Stat(cacheFilepath); err == nil {
		return cacheFilepath, nil
	}
	cacheFilepath, err = conn.Get(bucket, key)
	if err != nil {
		log.Debugln("Failed to cachedGet key: ", key)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}

func (conn *S3Connection) Get(bucket string, key string) (string, error) {
	cacheFilepath, err := conn.getCacheFilepath(key)
	if err != nil {
		log.Debugf("Failed to getCacheFilepath in Get: %s", err)
		return cacheFilepath, err
	}
	err = os.MkdirAll(path.Dir(cacheFilepath), 0777)
	if err != nil {
		log.Debugf("Couldn't create cache directory for cacheFilepath %s: %s",
			cacheFilepath, err)
		return cacheFilepath, err
	}
	w, err := os.Create(cacheFilepath)
	if err != nil {
		log.Debugf("Couldn't create cache file for cacheFilepath %s: %s",
			cacheFilepath, err)
		return cacheFilepath, err
	}
	defer w.Close()
	_, err = conn.Downloader.Download(w, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	time.Sleep(100 * time.Millisecond)
	if err != nil {
		log.Debugf("Failed to download key: %s", err)
		defer os.Remove(cacheFilepath)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}
