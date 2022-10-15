FROM webrecorder/browsertrix-crawler
RUN apt update && apt install httrack -y
