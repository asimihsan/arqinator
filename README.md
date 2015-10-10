# arqinator

Cross-platform restoration of Arq backups.

**This utility is not production ready. Please explicitly test it before assuming it will work for you.**

## Getting started

### 0. Prerequisites

How to set environment variables on:

-   Linux: https://www.digitalocean.com/community/tutorials/how-to-read-and-set-environmental-and-shell-variables-on-a-linux-vps
-   Mac OS X: http://apple.stackexchange.com/questions/106778/how-do-i-set-environment-variables-on-os-x
-   Windows: https://www.microsoft.com/resources/documentation/windows/xp/all/proddocs/en-us/sysdm_advancd_environmnt_addchange_variable.mspx?mfr=true

### 1. Configure Credentials

For all backup methods, set the following environment variable to the password
you're using for encrypting your Arq backups:

```
ARQ_ENCRYPTION_PASSWORD=mysecretpassword
```

Only Arq backup sets encrypted by this password will be visible to you when you
run `list-backup-sets`. If the password is incorrect they will appear to be
invisible. Run with `--verbose` if you want more information.

#### S3

Set the following environment variables depending on which backup method you
are using:

```
AWS_ACCESS_KEY_ID=AKID1234567890
AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

#### Google Cloud Storage

-   Go to the Google Developers Console: https://console.developers.google.com//project/_/apiui/credential
-   Download a JSON private key file using: https://goo.gl/SK5Rb7
-   To determine your project ID, click on the gear in the top right, and click "Project information"
-   To determine your bucket name, on the left navigation pane go to Storage - Cloud Storage - Browser, then find your bucket.

### 2. List backup sets

#### S3

```
$ arqinator \
    --backup-type s3 \
    --s3-region us-west-2 \
    --s3-bucket-name arq-akiajmthnhpkz2ixzrxq-us-west-2 \
    list-backup-sets

ArqBackupSet
    UUID 98DB38F8-B9C6-4296-9385-3C1BF858ED5D
    ComputerName Mill
    UserName ai
    Folders
        LocalPath /Users/ai
        UUID 8D4FAD2A-9E08-46F7-829D-E9601A65455D
```

#### Google Cloud Storage

```
$ arqinator \
    --backup-type googlecloudstorage \
    --gcs-json-private-key-filepath /Users/ai/keys/gcs.json \
    --gcs-project-id midyear-courage-109219 \
    --gcs-bucket-name arq-560729839528 \
    list-backup-sets

ArqBackupSet
    UUID 7FE8D069-B218-4E17-8E58-0C7FAF8CFAFC
    ComputerName Mill
    UserName ai
    Folders
        LocalPath /Users/ai/temp/apsw-3.7.15.1-r1
        UUID E6F4BC5E-B21F-4828-ADCC-8521F9DBC4C9
```

### 3. List directory contents of backups

#### S3

```
$ arqinator \
    --backup-type s3 \
    --s3-region us-west-2 \
    --s3-bucket-name arq-akiajmthnhpkz2ixzrxq-us-west-2 \
    list-directory-contents \
    --backup-set-uuid 98DB38F8-B9C6-4296-9385-3C1BF858ED5D \
    --folder-uuid 8D4FAD2A-9E08-46F7-829D-E9601A65455D \
    --path /Users/ai/.ssh

-rwx------	2013-06-29 05:43:06 -0700 PDT	1.7kB	ai_keypair_3.pem
-rw-r--r--	2013-06-28 13:44:47 -0700 PDT	0B	    config
-rw-------	2014-12-31 19:41:47 -0800 PST	1.7kB	digitalocean
-rw-r--r--	2014-12-31 19:41:47 -0800 PST	402B	digitalocean.pub
-rw-------	2012-04-11 12:49:10 -0700 PDT	1.7kB	id_rsa
-rw-r--r--	2012-04-14 13:16:50 -0700 PDT	396B	id_rsa.pub
-rw-------	2014-05-17 05:04:13 -0700 PDT	1.7kB	interview-ec2.pem
-rw-r--r--	2015-09-08 14:09:39 -0700 PDT	18kB	known_hosts
```

#### Google Cloud Storage

```
$ arqinator \
    --backup-type googlecloudstorage \
    --gcs-json-private-key-filepath /Users/ai/keys/gcs.json \
    --gcs-project-id midyear-courage-109219 \
    --gcs-bucket-name arq-560729839528 \
    list-directory-contents \
    --backup-set-uuid 7FE8D069-B218-4E17-8E58-0C7FAF8CFAFC \
    --folder-uuid E6F4BC5E-B21F-4828-ADCC-8521F9DBC4C9 \
    --path /Users/ai/temp/apsw-3.7.15.1-r1

-rw-r--r--	2015-10-08 12:36:21 -0700 PDT	6.1kB	.DS_Store
-rw-r--r--	2010-01-05 14:53:28 -0800 PST	699B	MANIFEST.in
-rw-rw-r--	2012-12-22 02:57:02 -0800 PST	1.0kB	PKG-INFO
drwxr-xr-x	2015-10-08 12:36:21 -0700 PDT	9.3MB	build
-rw-r--r--	2012-12-22 01:05:48 -0800 PST	7.1kB	checksums
drwxr-xr-x	2012-12-26 09:56:54 -0800 PST	1.6MB	doc
-rw-r--r--	2009-09-12 21:50:06 -0700 PDT	4.3kB	mingwsetup.bat
-rw-rw-r--	2012-12-26 10:01:38 -0800 PST	33kB	setup.py
drwxr-xr-x	2012-12-26 10:02:47 -0800 PST	7.6MB	sqlite3
drwxr-xr-x	2012-12-26 10:02:47 -0800 PST	554kB	src
-rw-r--r--	2012-12-26 10:05:00 -0800 PST	29kB	testdbx
-rw-r--r--	2012-12-26 10:05:00 -0800 PST	12kB	testdbx-journal
-rw-r--r--	2012-12-22 01:36:24 -0800 PST	340kB	tests.py
-rw-r--r--	2012-12-26 10:03:31 -0800 PST	285kB	tests.pyc
drwxr-xr-x	2012-12-26 09:56:54 -0800 PST	168kB	tools
```


### 4. Restore

You can restore either individual files or entire folders.

#### S3

```
$ arqinator \
    --backup-type s3 \
    --s3-region us-west-2 \
    --s3-bucket-name arq-akiajmthnhpkz2ixzrxq-us-west-2 \
    recover \
    --backup-set-uuid 98DB38F8-B9C6-4296-9385-3C1BF858ED5D \
    --folder-uuid 8D4FAD2A-9E08-46F7-829D-E9601A65455D \
    --source-path /Users/ai/output.txt \
    --destination-path /Users/ai/temp/output.txt
```

#### Google Cloud Storage

````
$ arqinator \
    --backup-type googlecloudstorage \
    --gcs-json-private-key-filepath /Users/ai/keys/gcs.json \
    --gcs-project-id midyear-courage-109219 \
    --gcs-bucket-name arq-560729839528 \
    recover \
    --backup-set-uuid 7FE8D069-B218-4E17-8E58-0C7FAF8CFAFC \
    --folder-uuid E6F4BC5E-B21F-4828-ADCC-8521F9DBC4C9 \
    --source-path /Users/ai/temp/apsw-3.7.15.1-r1/PKG-INFO \
    --destination-path /Users/ai/temp/PKG-INFO
````

## TODO

-   soft links
-   do you want to 'chown' files and folders to the UID/GID backed up?
-   support multiple encryption passwords for multiple accounts
    -   maybe have a text-file based configuration?
-   support all backup types possible with Arq, start with SFTP.
-   explicitly check SHA1 hashes of blobs to confirm no corruption.

### Bugs

-   Implement un-cached get, so that we can get the latest SHA from the
    master commit without having to delete the cache.

## How to do a release

https://github.com/aktau/github-release

```
github-release release \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "v0.1.0 test release" \
    --description "Initial release" \
    --pre-release

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "arqinator-osx-386.gz" \
    --file build/mac32/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "arqinator-osx-amd64.gz" \
    --file build/mac64/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "arqinator-linux-386.gz" \
    --file build/linux32/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "arqinator-linux-amd64.gz" \
    --file build/linux64/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "arqinator-windows-386.gz" \
    --file build/windows32/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0 \
    --name "arqinator-windows-amd64.gz" \
    --file build/windows64/arqinator.gz
```

Or if you need to delete a release:

```
github-release delete \
    --user asimihsan \
    --repo arqinator \
    --tag v0.1.0
```
