/*
arqinator: arq/types/main.go
Implements command-line interface to restoring Arq backups.

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
)

func cliSetup(c *cli.Context) error {
	switch c.GlobalString("backup-type") {
	case "s3":
	case "googlecloudstorage":
	default:
		return errors.New("Currently only support backup-type of: ['s3', 'googlecloudstorage']")
	}
	if c.GlobalBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func awsSetup(c *cli.Context) (connector.Connection, error) {
	region := c.GlobalString("s3-region")
	s3BucketName := c.GlobalString("s3-bucket-name")
	cacheDirectory := c.GlobalString("cache-directory")

	defaults.DefaultConfig.Region = aws.String(region)
	svc := s3.New(nil)
	opts := &s3manager.DownloadOptions{
		S3:          svc,
		Concurrency: runtime.GOMAXPROCS(0),
	}
	s3Connection := connector.NewS3Connection(svc, cacheDirectory, s3BucketName, opts)
	return s3Connection, nil
}

func googleCloudStorageSetup(c *cli.Context) (connector.Connection, error) {
	jsonPrivateKeyFilepath := c.GlobalString("gcs-json-private-key-filepath")
	projectID := c.GlobalString("gcs-project-id")
	bucketName := c.GlobalString("gcs-bucket-name")
	cacheDirectory := c.GlobalString("cache-directory")

	connection, err := connector.NewGoogleCloudStorageConnection(jsonPrivateKeyFilepath, projectID, bucketName, cacheDirectory)
	if err != nil {
		return connection, err
	}
	return connection, nil
}

func getConnection(c *cli.Context) (connector.Connection, error) {
	var (
		connection connector.Connection
		err        error
	)
	switch c.GlobalString("backup-type") {
	case "s3":
		connection, err = awsSetup(c)
	case "googlecloudstorage":
		connection, err = googleCloudStorageSetup(c)
	}
	if err != nil {
		log.Debugf("%s", err)
		return nil, err
	}
	return connection, nil
}

func getArqBackupSets(c *cli.Context, connection connector.Connection) ([]*arq.ArqBackupSet, error) {
	password := []byte(os.Getenv("ARQ_ENCRYPTION_PASSWORD"))

	arqBackupSets, err := arq.GetArqBackupSets(connection, password)
	if err != nil {
		log.Debugf("Error during getArqBackupSets: %s", err)
		return nil, err
	}
	return arqBackupSets, nil
}

func listBackupSets(c *cli.Context, connection connector.Connection) error {
	arqBackupSets, err := getArqBackupSets(c, connection)
	if err != nil {
		log.Debugf("Error during listBackupSets: %s", err)
		return nil
	}
	for _, arqBackupSet := range arqBackupSets {
		fmt.Printf("ArqBackupSet\n")
		fmt.Printf("    UUID %s\n", arqBackupSet.UUID)
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

func findBucket(c *cli.Context, connection connector.Connection, backupSetUUID string, folderUUID string) (*arq.ArqBucket, error) {
	arqBackupSets, err := getArqBackupSets(c, connection)
	if err != nil {
		log.Debugf("Error during findBucket: %s", err)
		return nil, err
	}
	var bucket *arq.ArqBucket
	for _, arqBackupSet := range arqBackupSets {
		if arqBackupSet.UUID == backupSetUUID {
			for _, folder := range arqBackupSet.Buckets {
				if folder.UUID == folderUUID {
					bucket = folder
				}
			}
		}
	}
	if bucket == nil {
		err := errors.New(fmt.Sprintf("Couldn't find backup set UUID %s, folder UUID %s.", backupSetUUID, folderUUID))
		log.Errorf("%s", err)
		return nil, err
	}
	return bucket, nil
}

func listDirectoryContents(c *cli.Context, connection connector.Connection) error {
	cacheDirectory := c.GlobalString("cache-directory")
	backupSetUUID := c.String("backup-set-uuid")
	folderUUID := c.String("folder-uuid")
	targetPath := c.String("path")

	bucket, err := findBucket(c, connection, backupSetUUID, folderUUID)
	if err != nil {
		err := errors.New(fmt.Sprintf("Couldn't find backup set UUID %s, folder UUID %s.", backupSetUUID, folderUUID))
		log.Errorf("%s", err)
		return err
	}
	log.Printf("Caching tree pack sets. If this is your first run, will take a few minutes...")
	backupSet := bucket.ArqBackupSet
	backupSet.CacheTreePackSets()
	log.Printf("Cached tree pack sets.")

	tree, node, err := arq.FindNode(cacheDirectory, backupSet, bucket, targetPath)
	if err != nil {
		log.Errorf("Failed to find target path %s: %s", targetPath, err)
		return err
	}
	if node == nil || node.IsTree.IsTrue() {
		if tree == nil {
			err2 := errors.New(fmt.Sprintf("node is tree but no tree found: %s", node))
			log.Errorf("%s", err2)
			return err2
		}
		apsi, _ := arq.NewPackSetIndex(cacheDirectory, backupSet, bucket)
		for _, node := range tree.Nodes {
			if node.IsTree.IsTrue() {
				tree, err := apsi.GetPackFileAsTree(backupSet, bucket, *node.DataBlobKeys[0].SHA1)
				if err != nil {
					log.Debugf("Failed to find tree for node %s: %s", node, err)
					node.PrintOutput()
				} else if tree == nil {
					log.Debugf("directory node %s has no tree", node)
					node.PrintOutput()
				} else {
					tree.PrintOutput(node)
				}
			} else {
				node.PrintOutput()
			}
		}
	} else {
		node.PrintOutput()
	}
	return nil
}

func recover(c *cli.Context, connection connector.Connection) error {
	cacheDirectory := c.GlobalString("cache-directory")
	backupSetUUID := c.String("backup-set-uuid")
	folderUUID := c.String("folder-uuid")
	sourcePath := c.String("source-path")
	destinationPath := c.String("destination-path")

	if _, err := os.Stat(destinationPath); err == nil {
		err := errors.New(fmt.Sprintf("Destination path %s already exists, won't overwrite.", destinationPath))
		log.Errorf("%s", err)
		return err
	}
	bucket, err := findBucket(c, connection, backupSetUUID, folderUUID)
	if err != nil {
		err := errors.New(fmt.Sprintf("Couldn't find backup set UUID %s, folder UUID %s.", backupSetUUID, folderUUID))
		log.Errorf("%s", err)
		return err
	}
	log.Printf("Caching tree and blob pack sets. If this is your first run, will take a few minutes...")
	backupSet := bucket.ArqBackupSet
	backupSet.CacheTreePackSets()
	backupSet.CacheBlobPackSets()
	log.Printf("Cached tree and blob pack sets.")

	tree, node, err := arq.FindNode(cacheDirectory, backupSet, bucket, sourcePath)
	if err != nil {
		log.Errorf("Failed to find source path %s: %s", sourcePath, err)
		return err
	}
	if node.IsTree.IsTrue() {
		err = arq.DownloadTree(tree, cacheDirectory, backupSet, bucket, sourcePath, destinationPath)
	} else {
		err = arq.DownloadNode(node, cacheDirectory, backupSet, bucket, sourcePath, destinationPath)
	}
	if err != nil {
		log.Errorf("recover failed to download node: %s", err)
		return err
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
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "backup-type",
			Usage: "Method used for backup, one of: ['s3', 'googlecloudstorage']",
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
			Name:  "gcs-json-private-key-filepath",
			Usage: "Google Cloud Storage JSON private key filepath. See: https://goo.gl/SK5Rb7",
		},
		cli.StringFlag{
			Name:  "gcs-project-id",
			Usage: "Google Cloud Storage project ID.",
		},
		cli.StringFlag{
			Name:  "gcs-bucket-name",
			Usage: "Google Cloud Storage bucket name.",
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
					log.Errorf("%s", err)
					return
				}
				connection, err := getConnection(c)
				if err != nil {
					log.Errorf("%s", err)
					return
				}
				if err := listBackupSets(c, connection); err != nil {
					log.Errorf("%s", err)
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
					log.Errorf("%s", err)
					return
				}
				connection, err := getConnection(c)
				if err != nil {
					log.Errorf("%s", err)
					return
				}
				if err := listDirectoryContents(c, connection); err != nil {
					log.Errorf("%s", err)
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
					log.Errorf("%s", err)
					return
				}
				connection, err := getConnection(c)
				if err != nil {
					log.Errorf("%s", err)
					return
				}
				if err := recover(c, connection); err != nil {
					log.Errorf("%s", err)
					return
				}
			},
		},
	}
	app.Run(os.Args)
}
