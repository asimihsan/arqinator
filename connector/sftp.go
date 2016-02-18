/*
arqinator: arq/types/sftp.go
Implements SFTP backup type for Arq.

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

/*
References:
-	http://stackoverflow.com/questions/31554196/ssh-connection-timeout
-	https://github.com/pkg/sftp/blob/master/examples/streaming-read-benchmark/main.go
-	https://github.com/paulstuart/sshclient/blob/master/client.go
*/

package connector

import (
	"fmt"
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	TIMEOUT    time.Duration = time.Duration(10 * time.Second)
	KEEPALIVE_INTERVAL time.Duration = time.Duration(3 * time.Second)
)

type SSHConn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *SSHConn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *SSHConn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

func DialSSHTimeout(network, addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}
	timeoutConn := &SSHConn{conn, timeout, timeout}
	c, chans, reqs, err := ssh.NewClientConn(timeoutConn, addr, config)
	if err != nil {
		return nil, err
	}
	client := ssh.NewClient(c, chans, reqs)

	// this sends keepalive packets every 3 seconds
	// there's no useful response from these, so we can just abort if there's an error
	go func() {
		t := time.NewTicker(KEEPALIVE_INTERVAL)
		defer t.Stop()
		for {
			<-t.C
			_, _, err := client.Conn.SendRequest("keepalive@golang.org", true, nil)
			if err != nil {
				log.Fatalf("Remote server did not respond to keepalive.")
				return
			}
		}
	}()
	return client, nil
}

type keychain struct {
	keys []ssh.Signer
}

func (k *keychain) PrivateKey(file string) error {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return err
	}
	k.keys = append(k.keys, key)
	return nil
}

func KeyAuth(file string) (ssh.AuthMethod, error) {
	k := new(keychain)
	if err := k.PrivateKey(file); err != nil {
		return nil, err
	}
	return ssh.PublicKeys(k.keys...), nil
}

type SFTPConnection struct {
	SSHClient      *ssh.Client
	SFTPClient     *sftp.Client
	RemotePath     string
	CacheDirectory string
}

func (c SFTPConnection) String() string {
	return fmt.Sprintf("{SFTPConnection: CacheDirectory=%s", c.CacheDirectory)
}

func (c SFTPConnection) Close() error {
	if c.SFTPClient != nil {
		if err := c.SFTPClient.Close(); err != nil {
			return err
		}
	}
	if c.SSHClient != nil {
		if err := c.SSHClient.Close(); err != nil {
			return err
		}
	}
	log.Debugf("SFTPConnection closed successfully.")
	return nil
}

func NewSFTPConnection(host string, port int, remotePath string, username string, password *string,
	privateKeyFilepath *string, cacheDirectory string) (*SFTPConnection, error) {
	log.Debugf("NewSFTPConnection entry. host: %s, port: %d, remotePath: %s, username: %s, privateKeyFilepath: %s, cacheDirectory: %s",
		host, port, remotePath, username, privateKeyFilepath, cacheDirectory)
	var (
		auths []ssh.AuthMethod
		err   error
	)
	if password != nil {
		log.Debugf("NewSFTPConnection plaintext password is present")
		auths = append(auths, ssh.Password(*password))
	}
	if privateKeyFilepath != nil {
		log.Debugf("NewSFTPConnection private key filepath not nil: %s", *privateKeyFilepath)
		keys, err := KeyAuth(*privateKeyFilepath)
		if err != nil {
			return nil, err
		}
		auths = append(auths, keys)
	}
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}
	config := &ssh.ClientConfig{
		User: username,
		Auth: auths,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	conn := SFTPConnection{
		RemotePath:     remotePath,
		CacheDirectory: cacheDirectory,
	}
	if conn.SSHClient, err = DialSSHTimeout("tcp", addr, config, TIMEOUT); err != nil {
		return nil, err
	}
	if conn.SFTPClient, err = sftp.NewClient(conn.SSHClient); err != nil {
		return nil, err
	}
	return &conn, nil
}

func (conn SFTPConnection) getCacheFilepath(key string) (string, error) {
	cacheFilepath := filepath.Join(conn.GetCacheDirectory(), key)
	cacheFilepath, err := filepath.Abs(cacheFilepath)
	if err != nil {
		log.Debugf("Failed to make cacheFilepath %s absolute: %s",
			cacheFilepath, err)
		return "", err
	}
	return cacheFilepath, nil
}

func (conn SFTPConnection) CachedGet(key string) (string, error) {
	cacheFilepath, err := conn.getCacheFilepath(key)
	if err != nil {
		log.Debugf("Failed to getCacheFilepath in CachedGet: %s", err)
		return "", err
	}
	log.Debugf("CachedGet key: %s, cacheFilepath: %s", key, cacheFilepath)
	fileInfo, err := os.Stat(cacheFilepath)
	if err == nil && fileInfo.Size() != 0 {
		// file exists, so if it's zero-byte then we don't need to retrieve it again
		// however the file could still be corrupted. a connector cannot know if a file is corrupted or not,
		// it's up to callers to verify that downloaded files are uncorrupted.
		return cacheFilepath, nil
	}
	cacheFilepath, err = conn.Get(key)
	if err != nil {
		log.Debugln("Failed to cachedGet key: ", key)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}

func (conn SFTPConnection) Get(key string) (string, error) {
	cacheFilepath, err := conn.getCacheFilepath(key)
	if err != nil {
		log.Errorf("Failed to getCacheFilepath in Get: %s", err)
		return cacheFilepath, err
	}
	cacheDirectory := filepath.Dir(cacheFilepath)
	if err = os.MkdirAll(cacheDirectory, 0777); err != nil {
		log.Errorf("Couldn't create cache directory %s for cacheFilepath %s: %s",
			cacheDirectory, cacheFilepath, err)
		return cacheFilepath, err
	}
	if _, err = os.Stat(cacheDirectory); err != nil {
		log.Errorf("Cache directory %s doesn't exist!", cacheDirectory)
		return cacheFilepath, err
	}
	w, err := os.Create(cacheFilepath)
	if err != nil {
		log.Errorf("Couldn't create cache file for cacheFilepath %s: %s",
			cacheFilepath, err)
		return cacheFilepath, err
	}
	defer w.Close()
	wBuffered := bufio.NewWriter(w)
	defer wBuffered.Flush()
	remoteFullpath := conn.SFTPClient.Join(conn.RemotePath, key)
	r, err := conn.SFTPClient.Open(remoteFullpath)
	if err != nil{
		log.Errorf("Failed to open remote file %s: %s", remoteFullpath, err)
		return cacheFilepath, err
	}
	defer r.Close()
	io.Copy(wBuffered, r)
	time.Sleep(100 * time.Millisecond)
	if err != nil {
		log.Errorf("Failed to download key: %s", err)
		defer os.Remove(cacheFilepath)
		return cacheFilepath, err
	}
	return cacheFilepath, nil
}

func (c SFTPConnection) GetCacheDirectory() string {
	return c.CacheDirectory
}

type SFTPObject struct {
	Fullpath string
}

func (o SFTPObject) String() string {
	return fmt.Sprintf("{SFTPObject: Fullpath=%s}", o.Fullpath)
}

func (o SFTPObject) GetPath() string {
	return o.Fullpath
}

func (conn SFTPConnection) ListObjectsAsFolders(prefix string) ([]Object, error) {
	return conn.listObjects(prefix)
}

func (conn SFTPConnection) ListObjectsAsAll(prefix string) ([]Object, error) {
	return conn.listObjects(prefix)
}

func (conn SFTPConnection) listObjects(key string) ([]Object, error) {
	remoteFullpath := conn.SFTPClient.Join(conn.RemotePath, key)
	files, err := conn.SFTPClient.ReadDir(remoteFullpath)
	if err != nil {
		return nil, err
	}
	objects := make([]Object, len(files))
	for i, file := range files {
		fileFullpath := conn.SFTPClient.Join(remoteFullpath, file.Name())
		fileFullpath = strings.TrimPrefix(fileFullpath, conn.RemotePath)
		fileFullpath = strings.TrimPrefix(fileFullpath, "/")
		object := SFTPObject{
			Fullpath: fileFullpath,
		}
		objects[i] = object
	}
	return objects, nil
}
