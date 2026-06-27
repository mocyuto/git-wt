#!/bin/bash
#
sudo chmod +x .githooks/pre-commit
git config core.hooksPath .githooks || true
sudo chown -R vscode:vscode /go
go mod download
