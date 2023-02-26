#!/bin/bash

# Make Linux / macOS vesrion
linux_macos_release=releases/linux-macos-site-archiving-toolkit
mkdir -p $linux_macos_release
cp -r resources $linux_macos_release
cp *.sh $linux_macos_release
rm $linux_macos_release/make-releases.sh
cd releases
zip -r linux-macos-site-archiving-toolkit.zip .

# Make Windows version
cd ..
windows_release=releases/windows-site-archiving-toolkit
mkdir -p $windows_release
cp -r resources $windows_release
cp *.ps1 $windows_release
cd releases
zip -r windows-site-archiving-toolkit.zip .