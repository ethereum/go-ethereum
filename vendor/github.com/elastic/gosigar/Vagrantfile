# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "hashicorp/precise64"
  config.vm.provision "shell", inline: "mkdir -p /home/vagrant/go"
  config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/cloudfoundry/gosigar"
  config.vm.provision "shell", inline: "chown -R vagrant:vagrant /home/vagrant/go"
  install_go = <<-BASH
  set -e

if [ ! -d "/usr/local/go" ]; then
	cd /tmp && wget https://storage.googleapis.com/golang/go1.3.3.linux-amd64.tar.gz
	cd /usr/local
	tar xvzf /tmp/go1.3.3.linux-amd64.tar.gz
	echo 'export GOPATH=/home/vagrant/go; export PATH=/usr/local/go/bin:$PATH:$GOPATH/bin' >> /home/vagrant/.bashrc
fi
export GOPATH=/home/vagrant/go
export PATH=/usr/local/go/bin:$PATH:$GOPATH/bin
/usr/local/go/bin/go get -u github.com/onsi/ginkgo/ginkgo
/usr/local/go/bin/go get -u github.com/onsi/gomega;
BASH
  config.vm.provision "shell", inline: 'apt-get install -y git-core'
  config.vm.provision "shell", inline: install_go
end
