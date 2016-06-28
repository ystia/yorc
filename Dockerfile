FROM loicalbertin/alpine-consul

# 0.6.16 is dynamically linked which causes problems (wait for 0.6.17 or 0.7.0)
ENV TERRAFORM_VERSION=0.6.15

RUN apk add --update ansible && \
    rm -rf /var/cache/apk/*

RUN cd /tmp && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    cd /usr/local/bin && \
    unzip /tmp/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    addgroup janus && \
    adduser -D -G janus janus && \
    mkdir -p /var/janus && \
    chown -R janus:janus /var/janus && \
    rm -fr /tmp/*

COPY ./janus /usr/local/bin/

ADD pkg/rootfs /

EXPOSE 8800
