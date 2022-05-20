#!/bin/bash

set -e -x

# -- Build for macOS and upload to Azure
go run build/ci.go install -dlgo
go run build/ci.go archive -type tar # -signer OSX_SIGNING_KEY -upload gethstore/builds

# # -- CocoaPods
# gem uninstall cocoapods -a -x
# gem install cocoapods
# mv ~/.cocoapods/repos/master ~/.cocoapods/repos/master.bak
# sed -i '.bak' 's/repo.join/!repo.join/g' $(dirname `gem which cocoapods`)/cocoapods/sources_manager.rb
# git clone --depth=1 https://github.com/CocoaPods/Specs.git ~/.cocoapods/repos/master
# pod setup --verbose

# -- Check XCode version
xctool -version
xcrun simctl list

# # -- Build for iOS and upload to Azure
# go run build/ci.go xcode -signer IOS_SIGNING_KEY -upload gethstore/builds
