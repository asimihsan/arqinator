/*
arqinator: arq/types/googlecloudstorage.go
Implements GCS backup type for Arq.

Copyright 2015 Asim Ihsan

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package connector

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
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

func (c GoogleCloudStorageConnection) Close() error {
	return nil
}

func getContext(jsonPrivateKeyFilepath string, projectID string) (context.Context, error) {
	jsonKey, err := ioutil.ReadFile(jsonPrivateKeyFilepath)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(jsonKey, storage.ScopeFullControl)
	if err != nil {
		return nil, err
	}
	ctx := cloud.NewContext(projectID, conf.Client(oauth2.NoContext))
	return ctx, nil
}

func NewGoogleCloudStorageConnection(jsonPrivateKeyFilepath string, projectID string, bucketName string,
	cacheDirectory string) (GoogleCloudStorageConnection, error) {
	context, err := getContext(jsonPrivateKeyFilepath, projectID)
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
		Prefix:    prefix,
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
		log.Errorf("Failed to getCacheFilepath in Get: %s", err)
		return cacheFilepath, err
	}
	cacheDirectory := filepath.Dir(cacheFilepath)
	err = os.MkdirAll(cacheDirectory, 0777)
	if err != nil {
		log.Errorf("Couldn't create cache directory for cacheFilepath %s: %s", cacheFilepath, err)
		return cacheFilepath, err
	}
	if _, err = os.Stat(cacheDirectory); err != nil {
		log.Errorf("Cache directory %s doesn't exist!", cacheDirectory)
		return cacheFilepath, err
	}
	w, err := os.Create(cacheFilepath)
	if err != nil {
		log.Errorf("Couldn't create cache file for cacheFilepath %s: %s", cacheFilepath, err)
		return cacheFilepath, err
	}
	defer w.Close()
	wBuffered := bufio.NewWriter(w)
	defer wBuffered.Flush()
	r, err := storage.NewReader(conn.Context, conn.BucketName, name)
	if err != nil {
		log.Errorf("Failed to download name %s during initialization: %s", name, err)
		defer os.Remove(cacheFilepath)
		return cacheFilepath, err
	}
	defer r.Close()
	_, err = io.Copy(wBuffered, r)
	time.Sleep(100 * time.Millisecond)
	if err != nil {
		log.Errorf("Failed to download name %s during download: %s", name, err)
		defer os.Remove(cacheFilepath)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}
