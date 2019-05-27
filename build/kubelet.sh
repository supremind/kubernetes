#!/bin/bash

set -ex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd ${DIR}/.. # project root path

./build/run.sh make WHAT=cmd/kubelet
