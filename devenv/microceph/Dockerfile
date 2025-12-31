FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y \
    snapd \
    fuse \
    squashfuse \
    systemd

COPY setup.sh /usr/local/bin/setup.sh
RUN chmod +x /usr/local/bin/setup.sh

COPY microceph-setup.service /etc/systemd/system/microceph-setup.service
RUN systemctl enable microceph-setup.service

STOPSIGNAL SIGRTMIN+3

ENTRYPOINT ["/sbin/init"]

