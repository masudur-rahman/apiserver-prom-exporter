FROM golang

COPY apiserver-prom-exporter /bin
#COPY . /go/src/github.com/masudur-rahman/apiserver-prom-exorter

#RUN go install /go/src/github.com/masudur-rahman/apiserver-prom-exporter

CMD ["start", "--bypass", "true", "--stopTime", "2"]
#ENTRYPOINT ["/go/bin/apiserver-prom-exporter"]
ENTRYPOINT ["/bin/apiserver-prom-exporter"]

EXPOSE 9999
