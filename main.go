package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mitchellh/go-homedir"

	"github.com/asimihsan/arqinator/arq"
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
	log.Printf("%x", pf[:100])

	//filepath, _ := abs.Connection.CachedGet(abs.S3BucketName, abs.Uuid+"/objects/"+"0418bf572b59518dadc0d383c8fc0a2c0011d91a")
	//data, _ := ioutil.ReadFile(filepath)
	//decrypted := abs.BlobDecrypter.Decrypt(data)
	//log.Println(string(decrypted))
	//new: 45979a8b411e46747343957240781e5ae14803ce
}
