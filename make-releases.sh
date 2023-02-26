#!/bin/bash

# Make Linux / macOS vesrion
linux-macos-release=releases/linux-macos-site-archiving-toolkit
mkdir -p $linux-macos-release
cp -r resources/ $linux-macos-release
cp *.sh $linux-macos-release
rm $linux-macos-release/make-releases.sh
cd releases
zip -r linux-macos-site-archiving-toolkit.zip .
rm $linux-macos-release

# Make Windows version
windows-release=releases/windows-site-archiving-toolkit
mkdir -p $windows-release
cp -r resources/ $windows-release
cp *.ps1 $windows-release
cd releases
zip -r windows-site-archiving-toolkit.zip .
rm $windows-release

