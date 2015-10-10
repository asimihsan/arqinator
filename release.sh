#!/usr/bin/env bash

#set -e
set -x

#make release

TAG=$1

github-release release \
    --user asimihsan \
    --repo arqinator \
    --tag $TAG \
    --pre-release

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-osx-386.gz" \
    --file build/mac32/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-osx-amd64.gz" \
    --file build/mac64/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${1}" \
    --name "arqinator-linux-386.gz" \
    --file build/linux32/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-linux-amd64.gz" \
    --file build/linux64/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-windows-386.gz" \
    --file build/windows32/arqinator.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-windows-amd64.gz" \
    --file build/windows64/arqinator.gz
