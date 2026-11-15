#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p .deploy
: "${GO_BIN:=go}"
(cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO_BIN" build -trimpath -ldflags='-s -w' -o ../.deploy/ai-buddy .)
(cd frontend && npm run build)
case "$(tar --version)" in
  *bsdtar*) set -- --no-xattrs --no-acls --no-fflags ;;
  *) set -- ;;
esac
COPYFILE_DISABLE=1 tar "$@" --exclude='.git' --exclude='node_modules' --exclude='.env' --exclude='.deploy/access.txt' --exclude='.deploy/backups' --exclude='coverage' --exclude='*.out' --exclude='*.log' -czf /private/tmp/ai-buddy-release.tar.gz .
printf 'Release: /private/tmp/ai-buddy-release.tar.gz\n'
