#/usr/bin/env bash

set -e

function log() {
  echo "[+] $1"
}

function fatal() {
  echo "[!] $1"
  exit 1
}

function platform() {
  local  __resultvar=$1
  if [[ -f "/etc/yum.conf" ]]; then
    eval $__resultvar="centos"
  elif [[ -f "/etc/dpkg/dpkg.cfg" ]]; then
    eval $__resultvar="ubuntu"
  else
    fatal "Unknwon operating system"
  fi
}
platform OS

function package() {
  if [[ $OS = "ubuntu" ]]; then
    if dpkg --get-selections | grep --quiet $1; then
      log "$1 is already installed. skipping."
    else
      apt-get install $@ -y
    fi
  elif [[ $OS = "centos" ]]; then
    if rpm -qa | grep --quiet $1; then
      log "$1 is already installed. skipping."
    else
      yum install $@ -y
    fi
  fi
}

function detect_fpm_output() {
  if [[ $OS = "ubuntu" ]]; then
    export FPM_OUTPUT=deb
  elif [[ $OS = "centos" ]]; then
    export FPM_OUTPUT=rpm
  fi
}
detect_fpm_output

function gem_install() {
  if gem list | grep --quiet $1; then
    log "$1 is already installed. skipping."
  else
    gem install $@
  fi
}

function main() {
  if [[ $# -ne 1 ]]; then
    fatal "Usage: $0 <rocksdb_version>"
  else
    log "using rocksdb version: $1"
  fi

  if [[ -d /vagrant ]]; then
    if [[ $OS = "ubuntu" ]]; then
      package g++-4.7
      export CXX=g++-4.7

      # the deb would depend on libgflags2, but the static lib is the only thing
      # installed by make install
      package libgflags-dev

      package ruby-all-dev
    elif [[ $OS = "centos" ]]; then
      pushd /etc/yum.repos.d
      if [[ ! -f /etc/yum.repos.d/devtools-1.1.repo ]]; then
        wget http://people.centos.org/tru/devtools-1.1/devtools-1.1.repo
      fi
      package devtoolset-1.1-gcc --enablerepo=testing-1.1-devtools-6
      package devtoolset-1.1-gcc-c++ --enablerepo=testing-1.1-devtools-6
      export CC=/opt/centos/devtoolset-1.1/root/usr/bin/gcc
      export CPP=/opt/centos/devtoolset-1.1/root/usr/bin/cpp
      export CXX=/opt/centos/devtoolset-1.1/root/usr/bin/c++
      export PATH=$PATH:/opt/centos/devtoolset-1.1/root/usr/bin
      popd
      if ! rpm -qa | grep --quiet gflags; then
        rpm -i https://github.com/schuhschuh/gflags/releases/download/v2.1.0/gflags-devel-2.1.0-1.amd64.rpm
      fi

      package ruby
      package ruby-devel
      package rubygems
      package rpm-build
    fi
  fi
  gem_install fpm

  make static_lib
  make install INSTALL_PATH=package
  fpm \
    -s dir \
    -t $FPM_OUTPUT \
    -n rocksdb \
    -v $1 \
    --prefix /usr \
    --url http://rocksdb.org/ \
    -m rocksdb@fb.com \
    --license BSD \
    --vendor Facebook \
    --description "RocksDB is an embeddable persistent key-value store for fast storage." \
    package
}

main $@
