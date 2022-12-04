#!/bin/bash
set -e

PROJECT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

for file_name in pre-commit pre-push; do
  if [[ -f "$PROJECT_PATH/.git/hooks/$file_name" ]]; then
    mv $PROJECT_PATH/.git/hooks/$file_name $PROJECT_PATH/.git/hooks/$file_name.bak
  fi
  ln -sf $PROJECT_PATH/bin/$file_name $PROJECT_PATH/.git/hooks/$file_name
done
