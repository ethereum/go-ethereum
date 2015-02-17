#!/bin/sh

set -e

if [ "$USER" != "root" ]; then
	echo 'This script must be run as root.'
	exit 1
fi

echo 'Remounting root as read-write ------------------------------------------------'

mount -o remount,rw /

echo 'Installing Go and dependencies -----------------------------------------------'

apt-get update
apt-get install -y \
	golang-go g++ git pkg-config ubuntu-app-launch\
	qtbase5-private-dev qtdeclarative5-private-dev libqt5opengl5-dev
apt-get clean

echo 'Setting up environment for phablet user --------------------------------------'

echo 'export GOPATH=$HOME' >> ~phablet/.bash_profile

echo 'Fetching the qml package -----------------------------------------------------'

su -l phablet -c 'go get gopkg.in/qml.v0'

echo 'Installing the .desktop file for the particle example ------------------------'

APP_ID='gopkg.in.qml.particle-example'
cp ~phablet/src/gopkg.in/qml.v*/cmd/ubuntu-touch/particle.desktop ~phablet/.local/share/applications/$APP_ID.desktop

echo 'Building and launching particle example --------------------------------------'

su -l phablet -c 'cd $HOME/src/gopkg.in/qml.v0/examples/particle; go build'

echo 'Launching particle example ---------------------------------------------------'

su -l phablet -c "ubuntu-app-launch $APP_ID"
