#!/bin/bash

# Prep
$release_dir=releases/site-archiving-toolkit

# Make Linux / macOS vesrion
mkdir -p $release_dir
cp -r resources $release_dir
cp *.sh $release_dir
rm $release_dir/make-releases.sh
cd releases
zip -r LINUX-MACOS-site-archiving-toolkit.zip site-archiving-toolkit
cd ..
rm -rf $release_dir

# Make Windows version
mkdir -p $release_dir
cp -r resources $release_dir
cp *.ps1 $release_dir
cd releases
zip -r WINDOWS-site-archiving-toolkit.zip site-archiving-toolkit
cd ..
rm -rf $release_dir