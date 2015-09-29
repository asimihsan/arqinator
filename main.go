package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mitchellh/go-homedir"

	"github.com/asimihsan/arqinator/arq"
	"github.com/asimihsan/arqinator/arq/types"
	"github.com/asimihsan/arqinator/connector"
)

func main() {
	defaults.DefaultConfig.Region = aws.String("us-west-2")
	svc := s3.New(nil)
	opts := &s3manager.DownloadOptions{S3: svc, Concurrency: 4}
	s3BucketName := "arq-akiajmthnhpkz2ixzrxq-us-west-2"
	arqBackupSetUuid := "98DB38F8-B9C6-4296-9385-3C1BF858ED5D"
	cacheDirectory, err := homedir.Expand("~/.arqinator_cache")
	if err != nil {
		log.Fatal("Failed to get user's home dir: ", err)
	}
	password := []byte(os.Getenv("ARQ_ENCRYPTION_PASSWORD"))

	s3Connection := connector.NewS3Connection(svc, cacheDirectory, opts)
	abs, _ := arq.NewArqBackupSet(s3BucketName, s3Connection, arqBackupSetUuid, password)
	log.Println("ArqBackupSet: ", abs)
	abs.CacheTreePackSets()

	ab := abs.Buckets[0]
	apsi, _ := arq.NewPackSetIndex(cacheDirectory, abs, ab)
	pf, _ := apsi.GetPackFile(abs, ab, ab.HeadSHA1)
	commit, err := arq_types.ReadCommit(bytes.NewBuffer(pf))
	if err != nil {
		log.Printf("failed to parse commit: %s", err)
	}
	log.Printf("%s", commit)

	tree_packfile, _ := apsi.GetPackFile(abs, ab, commit.TreeBlobKey.SHA1)
	if err != nil {
		log.Printf("failed to get tree blob: %s", err)
	}
	if commit.TreeBlobKey.IsCompressed.IsTrue() {
		var b bytes.Buffer
		r, _ := gzip.NewReader(bytes.NewBuffer(tree_packfile))
		io.Copy(&b, r)
		r.Close()
		tree_packfile = b.Bytes()
	}

	log.Printf("tree_packfile: %x", tree_packfile[:100])
	tree, err := arq_types.ReadTree(bytes.NewBuffer(tree_packfile))
	if err != nil {
		log.Printf("failed to get tree: %s", err)
	}
	log.Printf("tree: %s", tree)
}
