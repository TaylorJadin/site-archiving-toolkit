FROM webrecorder/browsertrix-crawler:0.8.1
RUN apt update && apt install httrack zip -y
COPY resources/replay-template.html /replay-template.html
COPY resources/httrack.sh /httrack.sh
COPY resources/webrecorder.sh /webrecorder.sh