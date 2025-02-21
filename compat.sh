#!/usr/bin/env bash
set -e

readonly MIN_BASH_VERSION=5
readonly MIN_DOCKER_VERSION=26.0.0
readonly MIN_BUILDX_VERSION=0.13

### Exit with error message
die() {
	echo "$@" >&2
	exit 1
}

### Bail and instruct user on missing package to install for their platform
die_pkg() {
	local -r package=${1?}
	local -r version=${2?}
	local install_cmd
	case "$OSTYPE" in
		linux*)
			if command -v "apt" >/dev/null; then
				install_cmd="apt install ${package}"
			elif command -v "yum" >/dev/null; then
				install_cmd="yum install ${package}"
			elif command -v "pacman" >/dev/null; then
				install_cmd="pacman -Ss ${package}"
			elif command -v "emerge" >/dev/null; then
				install_cmd="emerge ${package}"
			elif command -v "nix-env" >/dev/null; then
				install_cmd="nix-env -i ${package}"
			fi
		;;
		bsd*)     install_cmd="pkg install ${package}" ;;
		darwin*)  install_cmd="port install ${package}" ;;
		*) die "Error: Your operating system is not supported" ;;
	esac
	echo "Error: ${package} ${version}+ does not appear to be installed." >&2
	[ -n "$install_cmd" ] && echo "Try: \`${install_cmd}\`"  >&2
	exit 1
}

### Check if actual binary version is >= minimum version
check_version(){
	local pkg="${1?}"
	local have="${2?}"
	local need="${3?}"
	local i ver1 ver2 IFS='.'
	[[ "$have" == "$need" ]] && return 0
	read -r -a ver1 <<< "$have"
	read -r -a ver2 <<< "$need"
	for ((i=${#ver1[@]}; i<${#ver2[@]}; i++));
		do ver1[i]=0;
	done
	for ((i=0; i<${#ver1[@]}; i++)); do
		[[ -z ${ver2[i]} ]] && ver2[i]=0
		((10#${ver1[i]} > 10#${ver2[i]})) && return 0
		((10#${ver1[i]} < 10#${ver2[i]})) && die_pkg "${pkg}" "${need}"
	done
}

### Check if required binaries are installed at appropriate versions
check_tools(){
	if [ -z "${BASH_VERSINFO[0]}" ] \
	|| [ "${BASH_VERSINFO[0]}" -lt "${MIN_BASH_VERSION}" ]; then
		die_pkg "bash" "${MIN_BASH_VERSION}"
	fi
	for cmd in "$@"; do
		command -v "$1" >/dev/null || die "Error: $cmd not found"
		case $cmd in
			docker)
				version=$(docker version -f '{{ .Server.Version }}')
				check_version "docker" "${version}" "${MIN_DOCKER_VERSION}"
			;;
			buildx)
				version=$(docker system info -f '{{ range .ClientInfo.Plugins }}{{ if eq .Name "buildx" }}{{join (split .Version "v") "" }}{{ end }}{{ end }}')
				check_version "buildx" "${version}" "${MIN_BUILDX_VERSION}"
			;;
		esac
	done
}

check_tools docker buildx;

docker info -f '{{ .DriverStatus }}' \
    | grep "io.containerd.snapshotter.v1" >/dev/null \
|| die "Error: Docker Engine is not using containerd for image storage"