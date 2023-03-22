# Site Archiving Toolkit

## What is this thing?

The Site Archiving Toolkit allows you to quickly and easily make both flattened HTML and web archive versions of websites. These scripts are a relatively easy to use command line interface for crawling sites using both [HTTrack](https://www.httrack.com) and [Browsertrix Crawler](https://github.com/webrecorder/browsertrix-crawler) (from the [Webrecorder](https://webrecorder.net) project) in Docker.

Check out this video to see what it does and how to use it:

[Site Archiving Toolkit - reclaim.tv](https://archive.reclaim.tv/w/qYeNBzUdDWDxWi8pFSLNB1)

Check ou this example to see the types of archives it makes: 

[archiving.ca.reclaim.cloud](https://archiving.ca.reclaim.cloud/)

These archives are zipped and easy to download, where they can be placed on just about any web server and made public! Here's an example of one in use:

[digciz.jadin.me](https://digciz.jadin.me)

### Features

- Crawl an entire site / domain for offline browsing, preservation, or whatever other purpose
- Crawls can run in the background after they’ve been started, even if you close your terminal
- Accepts multiple URLs at once, to queue up multiple crawl jobs (on Linux or macOS, this is not supported on the Windows version)
- Preview archived pages using a local web server
- Automatically creates zip files for easy download/upload

## How do I use it?

The Site Archiving Toolkit is designed first to be run on Reclaim Cloud, but can also be used on any computer that has Docker installed.

### Using the Site Archiving Toolkit on Reclaim Cloud

Install the Site Archiving Toolkit using the Marketplace. Open the terminal (either via SSH or the built-in Web SSH feature) to start crawling sites.

The `archive` command will start crawling a site. Here are some examples:

This will crawl all pages on the "url.com" domain over HTTPS:
```bash
archive https://url.com
```

You can give the archive command a list of URLs seperated by spaces, and it will crawl them sequentially:
```bash
archive https://url.com https://anotherurl.com
```

Once you start a crawl using the `archive` command, you no longer need to keep your terminal open, as it will run in the background. If you need to stop crawling a site use `quit-crawlers` which will quit all httrack or browsertrix crawler jobs:
```
quit-crawlers
``` 

### Previewing and Downloading your archived sites

Visit the environment URL of your Site Archiving Toolkit environment to see all completed and in-progress crawls. When they are finished you can view them and download them as zip files. If you need to delete crawls that were made previously and are no longer needed, you can find them in the `crawls` directory, located at `/root/site-archiving-toolkit/crawls`, which is also bookmarked in the Reclaim Cloud file manager.

### Using the Site Archiving Toolkit on your own computer

- Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Launch Docker Desktop
- Download the latest version of the Site Archiving Toolkit for your OS from the [releases page](https://github.com/TaylorJadin/site-archiving-toolkit/releases)
- Unzip the release and place it somewhere convenient (maybe your Home directory or Documents folder)
- Open the Terminal on macOS, or Powershell on Windows
- `cd` to the folder you unzipped the release into (ex: `cd ~/Documents/site-archiving-toolkit`)

#### From here you can run any of the following commands on macOS or Linux:

`./archive.sh` to archive sites

`./quit-crawlers.sh` to quit any in-progress crawls

`./attach.sh` to re-attach to an in-progress crawl. This is useful if you started one earlier and closed your terminal, and now you want to check back up on their status.

`./start-server.sh` to start a local web server so you can preview your achived sites. After running this command, open up a web browser and navigate to <http://localhost>

`./stop-server.sh` to stop the local web server

#### Similar commands are available on Windows when using Powershell:

`.\archive.ps1` to archive sites. Note that the Windows version only supports one URL at a time.

`.\quit-crawlers.ps1` to quit any in-progress crawls

`.\attach.ps1` to re-attach to an in-progress crawl. This is useful if you started one earlier and closed your terminal, and now you want to check back up on the status.

`.\start-server.ps1` to start a local web server so you can preview your achived sites. After running this command, open up a web browser and navigate to <http://localhost>

`.\stop-server.ps1` to stop the local web server
