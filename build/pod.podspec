Pod::Spec.new do |spec|
  spec.name         = 'Geth'
  spec.version      = '1.8.2'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ethereum/go-ethereum'
  spec.authors      = { 'erichin' => 'erichinbato@gmail.com' }

  spec.summary      = 'iOS Ethereum Client'
  spec.source       = { :git => 'https://github.com/ethereum/go-ethereum.git' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'bin/Geth.framework'
end
