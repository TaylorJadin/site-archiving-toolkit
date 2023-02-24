FROM webrecorder/browsertrix-crawler
RUN apt update && apt install httrack zip -y
COPY resources/replay-template.html /replay-template.html
COPY resources/post-crawl.sh /post-crawl.sh