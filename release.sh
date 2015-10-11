#!/usr/bin/env bash

#set -e
set -x

make release

TAG=$1

github-release release \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --pre-release

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-osx-386.${TAG}.tar.gz" \
    --file build/mac/386/arqinator.tar.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-osx-amd64.${TAG}.tar.gz" \
    --file build/mac/amd64/arqinator.tar.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-linux-386.${TAG}.tar.gz" \
    --file build/linux/386/arqinator.tar.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-linux-amd64.${TAG}.tar.gz" \
    --file build/linux/amd64/arqinator.tar.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-windows-386.${TAG}.tar.gz" \
    --file build/windows/386/arqinator.tar.gz

github-release upload \
    --user asimihsan \
    --repo arqinator \
    --tag "${TAG}" \
    --name "arqinator-windows-amd64.${TAG}.tar.gz" \
    --file build/windows/amd64/arqinator.tar.gz
