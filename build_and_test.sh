#!/usr/bin/env bash


set -euxo pipefail

docker build . -t galera-init:test && docker run -t galera-init:test
