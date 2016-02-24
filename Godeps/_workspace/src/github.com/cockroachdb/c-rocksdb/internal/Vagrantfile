Vagrant.configure("2") do |config|

  config.vm.provider "virtualbox" do |v|
    v.memory = 4096
    v.cpus = 2
  end

  config.vm.define "ubuntu14" do |box|
    box.vm.box = "ubuntu/trusty64"
  end

  config.vm.define "centos65" do |box|
    box.vm.box = "chef/centos-6.5"
  end

  config.vm.define "FreeBSD10" do |box|
    box.vm.guest = :freebsd
    box.vm.box = "robin/freebsd-10"
    # FreeBSD does not support 'mount_virtualbox_shared_folder', use NFS
    box.vm.synced_folder ".", "/vagrant", :nfs => true, id: "vagrant-root"
    box.vm.network "private_network", ip: "10.0.1.10"

    # build everything after creating VM, skip using --no-provision
    box.vm.provision "shell", inline: <<-SCRIPT
      pkg install -y gmake clang35
      export CXX=/usr/local/bin/clang++35
      cd /vagrant
      gmake clean
      gmake all OPT=-g
    SCRIPT
  end

end
