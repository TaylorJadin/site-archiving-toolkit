#!/bin/bash

# Prep
release_dir=releases/site-archiving-toolkit
rm -rf releases

# Make Linux / macOS vesrion
mkdir -p $release_dir
cp -r resources $release_dir
cp *.sh $release_dir
rm $release_dir/make-releases.sh
cd releases
zip -r Linux-macOS-site-archiving-toolkit.zip site-archiving-toolkit
cd ..
rm -rf $release_dir

# Make Windows version
mkdir -p $release_dir
cp -r resources $release_dir
cp *.ps1 $release_dir
cd releases
zip -r Windows-site-archiving-toolkit.zip site-archiving-toolkit
cd ..
rm -rf $release_dir