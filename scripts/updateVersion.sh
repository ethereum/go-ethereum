#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

echo "The version is of form - VersionMajor.VersionMinor.VersionPatch-VersionMeta"
echo "Let's take 0.3.4-beta as an example. Here:"
echo "* VersionMajor is - 0"
echo "* VersionMinor is - 3"
echo "* VersionPatch is - 4"
echo "* VersionMeta is  - beta"
echo ""
echo "Now, enter the new version step-by-step below:"

version=""

# VersionMajor
read -p "* VersionMajor: " VersionMajor
if [ -z "$VersionMajor" ]
then
    echo "VersionMajor cannot be NULL"
    exit -1
fi
version+=$VersionMajor

# VersionMinor
read -p "* VersionMinor: " VersionMinor
if [ -z "$VersionMinor" ]
then
    echo "VersionMinor cannot be NULL"
    exit -1
fi
version+="."$VersionMinor

# VersionPatch
read -p "* VersionPatch: " VersionPatch
if [ -z "$VersionPatch" ]
then
    echo "VersionPatch cannot be NULL"
    exit -1
fi
version+="."$VersionPatch

# VersionMeta (optional)
read -p "* VersionMeta (optional, press enter if not needed): " VersionMeta
if [[ ! -z "$VersionMeta" ]]
then
    version+="-"$VersionMeta
fi

echo ""
echo "New version is: $version"

# update version in  ../params/version.go
versionFile="${DIR}/../params/version.go"
sed -i '' "s% = .*// Major% = $VersionMajor // Major%g" $versionFile
sed -i '' "s% = .*// Minor% = $VersionMinor // Minor%g" $versionFile
sed -i '' "s% = .*// Patch% = $VersionPatch // Patch%g" $versionFile
sed -i '' "s% = .*// Version metadata% = \"$VersionMeta\" // Version metadata%g" $versionFile
gofmt -w $versionFile

echo ""
echo "Updating Version Done"

exit 0
