# arqinator

Cross-platform restoration of Arq backups

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

Set the following environment variables depending on which backup method you
are using:

#### S3

```
AWS_ACCESS_KEY_ID=AKID1234567890
AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

### 2. List backup sets

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

### 3. List directory contents of backups

### 4. Restore

