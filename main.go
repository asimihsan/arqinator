package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/codegangsta/cli"
	"github.com/mitchellh/go-homedir"

	"errors"
	"fmt"
	"github.com/asimihsan/arqinator/arq"
	"github.com/asimihsan/arqinator/connector"
	"runtime"
	"bufio"
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
		S3:          svc,
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
}

func findBucket(c *cli.Context, s3Connection *connector.S3Connection, backupSetUuid string, folderUuid string) (*arq.ArqBucket, error) {
	arqBackupSets, err := getArqBackupSets(c, s3Connection)
	if err != nil {
		log.Debugf("Error during findBucket: %s", err)
		return nil, err
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
		return nil, err
	}
	return bucket, nil
}

func listDirectoryContents(c *cli.Context, s3Connection *connector.S3Connection) error {
	cacheDirectory := c.GlobalString("cache-directory")
	backupSetUuid := c.String("backup-set-uuid")
	folderUuid := c.String("folder-uuid")
	targetPath := c.String("path")

	bucket, err := findBucket(c, s3Connection, backupSetUuid, folderUuid)
	if err != nil {
		err := errors.New(fmt.Sprintf("Couldn't find backup set UUID %s, folder UUID %s.", backupSetUuid, folderUuid))
		log.Errorf("%s", err)
		return err
	}
	backupSet := bucket.ArqBackupSet
	backupSet.CacheTreePackSets()

	tree, node, err := arq.FindNode(cacheDirectory, backupSet, bucket, targetPath)
	if err != nil {
		log.Errorf("Failed to find target path %s: %s", targetPath, err)
		return err
	}
	if node == nil || node.IsTree.IsTrue() {
		for _, node := range tree.Nodes {
			node.PrintOutput()
		}
	} else {
		node.PrintOutput()
	}
	return nil
}

func recover(c *cli.Context, s3Connection *connector.S3Connection) error {
	cacheDirectory := c.GlobalString("cache-directory")
	backupSetUuid := c.String("backup-set-uuid")
	folderUuid := c.String("folder-uuid")
	sourcePath := c.String("source-path")
	destinationPath := c.String("destination-path")

	if _, err := os.Stat(destinationPath); err == nil {
		err := errors.New(fmt.Sprintf("Destination path %s already exists, won't overwrite.", destinationPath))
		log.Errorf("%s", err)
		return err
	}
	bucket, err := findBucket(c, s3Connection, backupSetUuid, folderUuid)
	if err != nil {
		err := errors.New(fmt.Sprintf("Couldn't find backup set UUID %s, folder UUID %s.", backupSetUuid, folderUuid))
		log.Errorf("%s", err)
		return err
	}
	backupSet := bucket.ArqBackupSet
	backupSet.CacheTreePackSets()
	backupSet.CacheBlobPackSets()

	tree, node, err := arq.FindNode(cacheDirectory, backupSet, bucket, sourcePath)
	if err != nil {
		log.Errorf("Failed to find source path %s: %s", sourcePath, err)
		return err
	}
	if node == nil || node.IsTree.IsTrue() {
		log.Errorf("unsupported right now. tree: %s", tree)
		return nil
	} else {
		apsi, _ := arq.NewPackSetIndex(cacheDirectory, backupSet, bucket)
		f, err := os.Create(destinationPath)
		if err != nil {
			log.Errorf("Failed to open destinationPath %s: %s", destinationPath, err)
			return err
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		for _, dataBlobKey := range node.DataBlobKeys {
			log.Debugf("node dataBlobKey: %s", dataBlobKey)
			var contents []byte
			contents, err = apsi.GetBlobPackFile(backupSet, bucket, *dataBlobKey.SHA1)
			if err != nil {
				log.Debugf("Couldn't find data in packfile, look at objects.")
				contents, err = arq.GetDataBlobKeyContentsFromObjects(*dataBlobKey.SHA1, bucket)
				if err != nil {
					log.Debugf("Couldn't find data in objects either!")
					return err
				}
			}
			w.Write(contents)
		}
		w.Flush()
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
			Name:  "verbose",
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
			Name:  "list-directory-contents",
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
		{
			Name:  "recover",
			Usage: "Recover a file or directory from a backup",
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
					Name:  "source-path",
					Usage: "Path of directory or file in backup",
				},
				cli.StringFlag{
					Name:  "destination-path",
					Usage: "Path to recover directory or file into. Must not already exist.",
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
				if err := recover(c, s3Connection); err != nil {
					log.Debugf("%s", err)
					return
				}
			},
		},
	}
	app.Run(os.Args)
}
