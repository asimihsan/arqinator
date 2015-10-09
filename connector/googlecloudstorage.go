package connector

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
	"io/ioutil"
	"path/filepath"
	"time"
	"os"
	"path"
	"io"
	"strings"
)

type GoogleCloudStorageConnection struct {
	Context        context.Context
	BucketName     string
	CacheDirectory string
}

func (c GoogleCloudStorageConnection) String() string {
	return fmt.Sprintf("{GoogleCloudStorageConnection: BucketName: %s, CacheDirectory=%s",
		c.BucketName, c.CacheDirectory)
}

func (c GoogleCloudStorageConnection) GetCacheDirectory() string {
	return c.CacheDirectory
}

func getContext(jsonPrivateKeyFilepath string, projectId string) (context.Context, error) {
	jsonKey, err := ioutil.ReadFile(jsonPrivateKeyFilepath)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(jsonKey, storage.ScopeFullControl)
	if err != nil {
		return nil, err
	}
	ctx := cloud.NewContext(projectId, conf.Client(oauth2.NoContext))
	return ctx, nil
}

func NewGoogleCloudStorageConnection(jsonPrivateKeyFilepath string, projectId string, bucketName string,
	cacheDirectory string) (GoogleCloudStorageConnection, error) {
	context, err := getContext(jsonPrivateKeyFilepath, projectId)
	if err != nil {
		return GoogleCloudStorageConnection{}, err
	}
	conn := GoogleCloudStorageConnection{
		Context:        context,
		BucketName:     bucketName,
		CacheDirectory: cacheDirectory,
	}
	return conn, nil
}

type GoogleCloudStorageObject struct {
	Name string
}

func (o GoogleCloudStorageObject) String() string {
	return fmt.Sprintf("{GoogleCloudStorageObject: Name=%s}", o.GetPath())
}

func (o GoogleCloudStorageObject) GetPath() string {
	return o.Name
}

func (conn GoogleCloudStorageConnection) ListObjectsAsFolders(prefix string) ([]Object, error) {
	return conn.listObjects(prefix, "/")
}

func (conn GoogleCloudStorageConnection) ListObjectsAsAll(prefix string) ([]Object, error) {
	return conn.listObjects(prefix, ",")
}

func (conn GoogleCloudStorageConnection) listObjects(prefix string, delimiter string) ([]Object, error) {
	log.Debugf("GoogleCloudStorageConnection listObjects. prefix: %s, delimeter: %s", prefix, delimiter)
	objects := make([]Object, 0)
	query := &storage.Query{
		Prefix: prefix,
		Delimiter: delimiter,
	}
	for {
		gcsObjects, err := storage.ListObjects(conn.Context, conn.BucketName, query)
		if err != nil {
			return objects, err
		}
		if delimiter == "/" { // folders
			for _, prefix := range gcsObjects.Prefixes {
				name := strings.TrimSuffix(prefix, delimiter)
				object := GoogleCloudStorageObject{
					Name: name,
				}
				objects = append(objects, object)
			}
		} else { // regular files
			for _, gcsObject := range gcsObjects.Results {
				object := GoogleCloudStorageObject{
					Name: gcsObject.Name,
				}
				objects = append(objects, object)
			}
		}
		time.Sleep(100 * time.Millisecond)
		query = gcsObjects.Next
		if query == nil {
			break
		}
	}
	log.Debugf("GoogleCloudStorageConnection listObjects returns: %s", objects)
	return objects, nil
}

func (conn GoogleCloudStorageConnection) getCacheFilepath(key string) (string, error) {
	cacheFilepath := filepath.Join(conn.GetCacheDirectory(), key)
	cacheFilepath, err := filepath.Abs(cacheFilepath)
	if err != nil {
		log.Debugf("Failed to make cacheFilepath %s absolute: %s", cacheFilepath, err)
		return "", err
	}
	return cacheFilepath, nil
}

func (conn GoogleCloudStorageConnection) CachedGet(name string) (string, error) {
	cacheFilepath, err := conn.getCacheFilepath(name)
	if err != nil {
		log.Debugf("Failed to getCacheFilepath in CachedGet: %s", err)
		return "", err
	}
	if _, err := os.Stat(cacheFilepath); err == nil {
		return cacheFilepath, nil
	}
	cacheFilepath, err = conn.Get(name)
	if err != nil {
		log.Debugln("Failed to cachedGet key: ", name)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}

func (conn GoogleCloudStorageConnection) Get(name string) (string, error) {
	log.Debugf("GoogleCloudStorageConnection Get. name: %s", name)
	cacheFilepath, err := conn.getCacheFilepath(name)
	if err != nil {
		log.Debugf("Failed to getCacheFilepath in Get: %s", err)
		return cacheFilepath, err
	}
	err = os.MkdirAll(path.Dir(cacheFilepath), 0777)
	if err != nil {
		log.Debugf("Couldn't create cache directory for cacheFilepath %s: %s", cacheFilepath, err)
		return cacheFilepath, err
	}
	w, err := os.Create(cacheFilepath)
	if err != nil {
		log.Debugf("Couldn't create cache file for cacheFilepath %s: %s", cacheFilepath, err)
		return cacheFilepath, err
	}
	defer w.Close()
	r, err := storage.NewReader(conn.Context, conn.BucketName, name)
	if err != nil {
		log.Debugf("Failed to download name %s during initialization: %s", name, err)
		defer os.Remove(cacheFilepath)
		return cacheFilepath, err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	time.Sleep(100 * time.Millisecond)
	if err != nil {
		log.Debugf("Failed to download name %s during download: %s", name, err)
		defer os.Remove(cacheFilepath)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}
