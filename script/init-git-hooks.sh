#!/bin/bash
which go
if [[ $? != 0 ]]; then
  echo Please install golang first!
  exit 1
fi
which shfmt
if [[ $? != 0 ]]; then
  while true; do
    echo "install shfmt..."
    go install mvdan.cc/sh/v3/cmd/shfmt@latest
    if [[ $? = 0 ]]; then
      break
    fi
  done
  echo "Can't install shfmt with your machine, please fix it!"
  exit 1
fi

SYSTEM_NAME=$(uname)
if [[ "$SYSTEM_NAME" = "Linux" ]]; then
  SYSTEM_NAME=$(echo $(awk -F= '/^NAME/{print $2}' /etc/os-release) | tr -d '"' | awk '{print $1}')
fi

case $SYSTEM_NAME in
  Darwin)
    which brew
    if [[ $? != 0 ]]; then
      ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)" </dev/null 2>/dev/null
      if [[ $? != 0 ]]; then
        echo "Can't install brew, please fix it and try again!"
        exit 1
      fi
    fi
    which jq
    if [[ $? != 0 ]]; then
      brew install jq
      if [[ $? != 0 ]]; then
        echo "Can't install jq, please fix it and try again!"
        exit 1
      fi
    fi
    ;;
  Ubuntu)
    apt-get update
    which jq
    if [[ $? != 0 ]]; then
      apt-get install jq
      if [[ $? != 0 ]]; then
        echo "Can't install jq, please fix it and try again!"
        exit 1
      fi
    fi
    ;;
  *)
    echo "can't install jq for system $SYSTEM_NAME"
    exit 1
    ;;
esac

set -e

PROJECT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd .. && pwd)"

for file_name in pre-commit pre-push; do
  if [[ -f "$PROJECT_PATH/.git/hooks/$file_name" ]]; then
    mv $PROJECT_PATH/.git/hooks/$file_name $PROJECT_PATH/.git/hooks/$file_name.bak
  fi
  ln -sf $PROJECT_PATH/script/$file_name $PROJECT_PATH/.git/hooks/$file_name
done
