# --- build lokean ---
FROM registry.access.redhat.com/ubi8:8.2 AS builder
ENV GOPATH=/go
ENV D=/go/src/github.com/infrawatch/lokean

WORKDIR $D
COPY . $D/

COPY build/repos/opstools.repo /etc/yum.repos.d/opstools.repo
RUN dnf install qpid-proton-c-devel git golang --setopt=tsflags=nodocs -y && \
        dnf clean all && \
        go build -o lokean cmd/main.go && \
        mv lokean /tmp/

# --- end build, create lokean layer ---
FROM registry.access.redhat.com/ubi8:8.2

LABEL io.k8s.display-name="Lokean" \
      io.k8s.description="A component of the Service Telemetry Framework on the server side that ingests data from AMQP 1.x and forwards logs to Loki" \
      maintainer="Jaromir Wysoglad <jwysogla@redhat.com>"

COPY build/repos/opstools.repo /etc/yum.repos.d/opstools.repo
RUN yum install qpid-proton-c --setopt=tsflags=nodocs -y && \
        yum clean all && \
        rm -rf /var/cache/yum

COPY --from=builder /tmp/lokean /

ENTRYPOINT ["/lokean"]
