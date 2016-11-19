Pod::Spec.new do |spec|
  spec.name         = '{{.Name}}'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ubiq/go-ubiq'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Ethereum Client'
  spec.source       = { :git => 'https://github.com/ubiq/go-ubiq.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gubiq.framework'

	spec.prepare_command = <<-CMD
    curl https://gubiqstore.blob.core.windows.net/builds/gubiq-ios-all-{{.Version}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv gubiq-ios-all-{{.Version}}/Gubiq.framework Frameworks
    rm -rf gubiq-ios-all-{{.Version}}
  CMD
end
