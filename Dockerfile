# --- build lokean ---
FROM centos:8 AS builder
ENV GOPATH=/go
ENV D=/go/src/github.com/infrawatch/lokean

WORKDIR $D
COPY . $D/

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install qpid-proton-c-devel git golang --setopt=tsflags=nodocs -y && \
        yum clean all && \
        go build -o lokean cmd/main.go && \
        mv lokean /tmp/

# --- end build, create lokean layer ---
FROM centos:8

LABEL io.k8s.display-name="Lokean" \
      io.k8s.description="A component of the Service Telemetry Framework on the server side that ingests data from AMQP 1.x and forwards logs to Loki" \
      maintainer="Jaromir Wysoglad <jwysogla@redhat.com>"

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install qpid-proton-c --setopt=tsflags=nodocs -y && \
        yum clean all && \
        rm -rf /var/cache/yum

COPY --from=builder /tmp/lokean /

ENTRYPOINT ["/lokean"]
