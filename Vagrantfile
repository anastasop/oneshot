
Vagrant::Config.run do |config|
  config.vm.box = "precise32"
  config.vm.network :hostonly, "192.168.33.11"
#  config.vm.forward_port 8080, 8080

  config.vm.provision :shell do |shell|
    shell.inline = "echo i am the provisioner"
  end
end
