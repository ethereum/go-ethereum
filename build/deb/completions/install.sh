#!/bin/bash
set -eo pipefail

PROFILE="${HOME}/.bashrc"
SRC_DIR="$(dirname "$0")"
DEST_DIR="${HOME}/.bash_completion.d"
XDC_PATH="\${HOME}/XDPoSChain/build/bin"
CMD_STR="PROG=XDC source \${HOME}/.bash_completion.d/xdc.sh"

mkdir -p ${DEST_DIR}
cp ${SRC_DIR}/bash_autocomplete "${DEST_DIR}/xdc.sh"
cp ${SRC_DIR}/zsh_autocomplete "${DEST_DIR}/xdc.zsh"

if ! grep PATH ${PROFILE} | grep ${XDC_PATH} &>/dev/null; then
    echo "export PATH=\${PATH}:${XDC_PATH}" >>"${PROFILE}"
fi
if ! grep -q "${CMD_STR}" "${PROFILE}"; then
    echo "${CMD_STR}" >>"${PROFILE}"
fi

echo "please login again or run: source \${HOME}/.bashrc"
