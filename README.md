# arqinator

Cross-platform restoration of [Arq](https://www.arqbackup.com/) backups.

**This utility is not production ready. Please explicitly test it before assuming it will work for you.**

## Features

-   Cross-platform support for Windows, Mac OS X, and Linux.
    -   Tested on Windows 7 32-bit, Mac OS X Yosemite 10.10.5 64-bit,
        and Ubuntu 14.04 LTS 64-bit
-   Deployable as a single executable file, no external dependencies required.
-   Recover single files, sub-folders and their contents, or entire backup
    sets.

## Limitations

-   Currently only supports the following backup types:
    -   S3
    -   Google Cloud Storage
    -   SFTP (only unencrypted SSH private keys)
-   arqinator has been tested on backups created by Arq 4.14.5 only. I do not
    know if arqinator works on previous versions of Arq. I'm doubtful that
    arqinator will work on previous major versions of Arq (i.e. 3 or 2).

I've successfully listed backup sets, listed directory contents, and downloaded
specific files and subfolders on:
 
-   Backups created by Mac OS X 10.10.5 onto S3, Google Cloud Storage, and SFTP,
    and then retrieved back onto the same Mac OS X host.
-   Backups created by Windows 7 onto S3, and then retrieved back onto the same Windows 7 host and a different
    Ubuntu 14.04 host.

## Requirements



## Getting started

### 0. Prerequisites

How to set environment variables on:

-   Linux: https://www.digitalocean.com/community/tutorials/how-to-read-and-set-environmental-and-shell-variables-on-a-linux-vps
-   Mac OS X: http://apple.stackexchange.com/questions/106778/how-do-i-set-environment-variables-on-os-x
-   Windows: https://www.microsoft.com/resources/documentation/windows/xp/all/proddocs/en-us/sysdm_advancd_environmnt_addchange_variable.mspx?mfr=true

Download arqinator binaries from the [releases page](https://github.com/asimihsan/arqinator/releases).

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

#### SFTP

The preferred way of using SFTP is to use password-less SSH login by putting your
SSH public key into the SFTP server's `authorized_keys`. When you do so
currently arqinator only supported unencrypted SSH private keys. However if
you want to log into the SFTP server using a plaintext password you can set
the following environment variable:

```
ARQ_SFTP_PASSWORD=my-sftp-password
```

### 2. List backup sets

Note that there will be a difference between how paths appear on Windows and
Linux/Mac:

#### S3, Mac

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

#### S3, Windows

```
arqinator ^
    --backup-type s3 ^
    --s3-region us-west-2 ^
    --s3-bucket-name arq-akiajmthnhpkz2ixzrxq-us-west-2 ^
    list-backup-sets

﻿ArqBackupSet
    UUID E7CFDEED-AB08-4970-A377-78F8313AC39C
    ComputerName THE_RAIN
    UserName SYSTEM
    Folders
        LocalPath /C/Users/username/Downloads/apsw-3.7.15.1-r1
        UUID FE8BE3EE-B63B-4D1F-A7E9-6707297823B5
```

#### Google Cloud Storage, Mac

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

#### SFTP, Mac, verbose mode

```
$ arqinator \
      --backup-type sftp \
      --sftp-host asims-mac-mini.local \
      --sftp-port 22 \
      --sftp-remote-path /Users/aihsan/arq_backup \
      --sftp-username aihsan \
      --sftp-private-key-filepath /Users/ai/.ssh/id_rsa \
      --verbose \
      list-backup-sets

ArqBackupSet
    UUID 76A4E004-FCB9-47D7-B080-16A236439F5C
    ComputerName Mill
    UserName ai
    Folders
        LocalPath /Users/ai/temp/apsw-3.7.15.1-r1
        UUID 1BFC0BD6-9877-4562-9692-05EB3A5EF20C
```

### 3. List directory contents of backups

Again note that paths for Windows will look a little unusual.

#### S3, Mac

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

#### S3, Windows

```
arqinator ^
    --backup-type s3 ^
    --s3-region us-west-2 ^
    --s3-bucket-name arq-akiajmthnhpkz2ixzrxq-us-west-2 ^
    list-directory-contents ^
    --backup-set-uuid E7CFDEED-AB08-4970-A377-78F8313AC39C ^
    --folder-uuid FE8BE3EE-B63B-4D1F-A7E9-6707297823B5 ^
    --path /C/Users/username/Downloads/apsw-3.7.15.1-r1

﻿----------      2015-10-08 20:36:21 +0100 BST   6.1kB   .DS_Store
d---------      1970-01-01 00:00:00 +0000 GMT   9.3MB   build
----------      2012-12-22 09:05:48 +0000 GMT   7.1kB   checksums
d---------      1970-01-01 00:00:00 +0000 GMT   1.6MB   doc
----------      2010-01-05 22:53:28 +0000 GMT   699B    MANIFEST.in
----------      2009-09-13 05:50:06 +0100 BST   4.3kB   mingwsetup.bat
----------      2012-12-22 10:57:02 +0000 GMT   1.0kB   PKG-INFO
----------      2012-12-26 18:01:38 +0000 GMT   33kB    setup.py
d---------      1970-01-01 00:00:00 +0000 GMT   7.6MB   sqlite3
d---------      1970-01-01 00:00:00 +0000 GMT   554kB   src
----------      2012-12-26 18:05:00 +0000 GMT   29kB    testdbx
----------      2012-12-26 18:05:00 +0000 GMT   12kB    testdbx-journal
----------      2012-12-22 09:36:24 +0000 GMT   340kB   tests.py
----------      2012-12-26 18:03:31 +0000 GMT   285kB   tests.pyc
d---------      1970-01-01 00:00:00 +0000 GMT   168kB   tools
```

#### Google Cloud Storage, Mac

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

#### SFTP, Mac

```
arqinator \
    --backup-type sftp \
    --sftp-host asims-mac-mini.local \
    --sftp-port 22 \
    --sftp-remote-path /Users/aihsan/arq_backup \
    --sftp-username aihsan \
    --sftp-private-key-filepath /Users/ai/.ssh/id_rsa \
    --verbose
    list-directory-contents \
    --backup-set-uuid 76A4E004-FCB9-47D7-B080-16A236439F5C \
    --folder-uuid 1BFC0BD6-9877-4562-9692-05EB3A5EF20C \
    --path /Users/ai/temp/apsw-3.7.15.1-r1/build

-rw-r--r--	2015-10-09 09:50:34 -0700 PDT	10kB	.DS_Store
drwxr-xr-x	2012-12-26 10:03:31 -0800 PST	1.5MB	lib.macosx-10.4-x86_64-2.7
drwxr-xr-x	2015-10-09 09:50:34 -0700 PDT	7.8MB	temp.macosx-10.4-x86_64-2.7
```

### 4. Restore

You can restore either individual files or entire folders. Note that you need
to use a Linux-like directory path for Windows backups:

#### S3, Mac, recovering a single file

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

#### S3, Windows, recovering an entire folder in verbose mode

```
arqinator ^
    --backup-type s3 ^
    --s3-region us-west-2 ^
    --s3-bucket-name arq-akiajmthnhpkz2ixzrxq-us-west-2 ^
    --verbose ^
    recover ^
    --backup-set-uuid E7CFDEED-AB08-4970-A377-78F8313AC39C ^
    --folder-uuid FE8BE3EE-B63B-4D1F-A7E9-6707297823B5 ^
    --source-path /C/Users/username/Downloads/apsw-3.7.15.1-r1/tools ^
    --destination-path /C/temp/tools
```

#### Google Cloud Storage, Mac, recovering a single file

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

### SFTP, Mac, recovering a folder

```
$ arqinator \
    --backup-type sftp \
    --sftp-host asims-mac-mini.local \
    --sftp-port 22 \
    --sftp-remote-path /Users/aihsan/arq_backup \
    --sftp-username aihsan \
    --sftp-private-key-filepath /Users/ai/.ssh/id_rsa \
    recover \
    --backup-set-uuid 76A4E004-FCB9-47D7-B080-16A236439F5C \
    --folder-uuid 1BFC0BD6-9877-4562-9692-05EB3A5EF20C \
    --source-path /Users/ai/temp/apsw-3.7.15.1-r1/build \
    --destination-path /Users/ai/temp/foobar
```

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
