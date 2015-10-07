package main

import (
	"bytes"
	"compress/gzip"
	"io"
	log "github.com/Sirupsen/logrus"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/codegangsta/cli"
	"github.com/mitchellh/go-homedir"

	"github.com/asimihsan/arqinator/arq"
	"github.com/asimihsan/arqinator/connector"
	"errors"
	"fmt"
	"runtime"
	"github.com/asimihsan/arqinator/arq/types"
)

func cliSetup(c *cli.Context) error {
	if c.GlobalString("backup-type") != "s3" {
		return errors.New("Currently only support backup-type of 's3'")
	}
	if c.GlobalBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func awsSetup(c *cli.Context) (*connector.S3Connection, error) {
	region := c.GlobalString("s3-region")
	cacheDirectory := c.GlobalString("cache-directory")

	defaults.DefaultConfig.Region = aws.String(region)
	svc := s3.New(nil)
	opts := &s3manager.DownloadOptions{
		S3: svc,
		Concurrency: runtime.GOMAXPROCS(0)}
	s3Connection := connector.NewS3Connection(svc, cacheDirectory, opts)
	return s3Connection, nil
}

func getArqBackupSets(c *cli.Context, s3Connection *connector.S3Connection) ([]*arq.ArqBackupSet, error) {
	s3BucketName := c.GlobalString("s3-bucket-name")
	password := []byte(os.Getenv("ARQ_ENCRYPTION_PASSWORD"))

	arqBackupSets, err := arq.GetArqBackupSets(s3BucketName, s3Connection, password)
	if err != nil {
		log.Debugf("Error during getArqBackupSets: %s", err)
		return nil, err
	}
	return arqBackupSets, nil
}

func listBackupSets(c *cli.Context, s3Connection *connector.S3Connection) error {
	arqBackupSets, err := getArqBackupSets(c, s3Connection)
	if err != nil {
		log.Debugf("Error during listBackupSets: %s", err)
		return nil
	}
	for _, arqBackupSet := range arqBackupSets {
		fmt.Printf("ArqBackupSet\n")
		fmt.Printf("    UUID %s\n", arqBackupSet.Uuid)
		fmt.Printf("    ComputerName %s\n", arqBackupSet.ComputerInfo.ComputerName)
		fmt.Printf("    UserName %s\n", arqBackupSet.ComputerInfo.UserName)
		fmt.Printf("    Folders\n")
		for _, bucket := range arqBackupSet.Buckets {
			fmt.Printf("        LocalPath %s\n", bucket.LocalPath)
			fmt.Printf("        UUID %s\n", bucket.UUID)
		}
	}

	return nil
	//abs, _ := arq.NewArqBackupSet(s3BucketName, s3Connection, arqBackupSetUuid, password)
	//log.Debugln("ArqBackupSet: ", abs)
	//abs.CacheTreePackSets()
}

func listDirectoryContents(c *cli.Context, s3Connection *connector.S3Connection) error {
	cacheDirectory := c.GlobalString("cache-directory")
	backupSetUuid := c.String("backup-set-uuid")
	folderUuid := c.String("folder-uuid")

	arqBackupSets, err := getArqBackupSets(c, s3Connection)
	if err != nil {
		log.Debugf("Error during listBackupSets: %s", err)
		return nil
	}
	var bucket *arq.ArqBucket
	for _, arqBackupSet := range arqBackupSets {
		if arqBackupSet.Uuid == backupSetUuid {
			for _, folder := range arqBackupSet.Buckets {
				if folder.UUID == folderUuid {
					bucket = folder
				}
			}
		}
	}
	if bucket == nil {
		err := errors.New(fmt.Sprintf("Couldn't find backup set UUID %s, folder UUID %s.", backupSetUuid, folderUuid))
		log.Errorf("%s", err)
		return err
	}
	backupSet := bucket.ArqBackupSet
	backupSet.CacheTreePackSets()

	apsi, _ := arq.NewPackSetIndex(cacheDirectory, backupSet, bucket)
	pf, _ := apsi.GetPackFile(backupSet, bucket, bucket.HeadSHA1)
	commit, err := arq_types.ReadCommit(bytes.NewBuffer(pf))
	if err != nil {
		log.Debugf("failed to parse commit: %s", err)
	}
	log.Debugf("commit: %s", commit)

	log.Debugf("get tree_packfile...")
	tree_packfile, _ := apsi.GetPackFile(backupSet, bucket, *commit.TreeBlobKey.SHA1)
	if err != nil {
		log.Debugf("failed to get tree blob: %s", err)
	}
	log.Debugf("finished getting tree_packfile.")
	log.Debugf("decompress tree_packfile...")
	if commit.TreeBlobKey.IsCompressed.IsTrue() {
		var b bytes.Buffer
		r, _ := gzip.NewReader(bytes.NewBuffer(tree_packfile))
		io.Copy(&b, r)
		r.Close()
		tree_packfile = b.Bytes()
	}
	log.Debugf("finished decompressing tree_packfile.")

	log.Debugf("get tree...")
	tree, err := arq_types.ReadTree(bytes.NewBuffer(tree_packfile))
	if err != nil {
		log.Debugf("failed to get tree: %s", err)
	}
	log.Debugf("finished getting tree.")
	log.Debugf("tree: %s", tree)
	for _, node := range tree.Nodes {
		fmt.Printf("%s\n", node.Name)
	}

	return nil
}

func main() {
	defaultCacheDirectory, err := homedir.Expand("~/.arqinator_cache")
	if err != nil {
		log.Fatal("Failed to get user's home dir: ", err)
	}

	app := cli.NewApp()
	app.Name = "arqinator"
	app.Usage = "restore folders and files from Arq backups"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "backup-type",
			Usage: "Method used for backup, e.g. 's3'.",
		},
		cli.StringFlag{
			Name:  "s3-region",
			Usage: "AWS S3 region, e.g. 'us-west-2'.",
		},
		cli.StringFlag{
			Name:  "s3-bucket-name",
			Usage: "AWS S3 bucket name, e.g. 'arq-akiaabdefg-us-west-2'.",
		},
		cli.StringFlag{
			Name:  "cache-directory",
			Value: defaultCacheDirectory,
			Usage: fmt.Sprintf("Where to cache Arq files for browsing. Default: %s", defaultCacheDirectory),
		},
		cli.BoolFlag{
			Name:  "delete-cache-directory",
			Usage: "Delete cache directory before starting. Useful if seeing errors that could be due to truncated downloads.",
		},
		cli.BoolFlag{
			Name: "verbose",
			Usage: "Enable verbose logging",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "list-backup-sets",
			Usage: "List backup sets in this account.",
			Action: func(c *cli.Context) {
				if err := cliSetup(c); err != nil {
					log.Debugf("%s", err)
					return
				}
				s3Connection, err := awsSetup(c)
				if err != nil {
					log.Debugf("%s", err)
					return
				}
				if err := listBackupSets(c, s3Connection); err != nil {
					log.Debugf("%s", err)
					return
				}
			},
		},
		{
			Name: "list-directory-contents",
			Usage: "List contents of directory in backup.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "backup-set-uuid",
					Usage: "UUID of backup set. Use 'list-backup-sets' to determine this.",
				},
				cli.StringFlag{
					Name:  "folder-uuid",
					Usage: "UUID of folder. Use 'list-backup-sets' to determine this.",
				},
				cli.StringFlag{
					Name:  "path",
					Usage: "Path of directory or file in backup",
				},
			},
			Action: func(c *cli.Context) {
				if err := cliSetup(c); err != nil {
					log.Debugf("%s", err)
					return
				}
				s3Connection, err := awsSetup(c)
				if err != nil {
					log.Debugf("%s", err)
					return
				}
				if err := listDirectoryContents(c, s3Connection); err != nil {
					log.Debugf("%s", err)
					return
				}
			},
		},
	}
	app.Run(os.Args)
}
