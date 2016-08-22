FROM loicalbertin/alpine-consul

# 0.7.0 last stable version
ENV TERRAFORM_VERSION=0.7.0

RUN apk add --update ansible && \
    rm -rf /var/cache/apk/*
# TODO join those 2 RUN (currently not joined for build/experiment raisons)
RUN cd /tmp && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    cd /usr/local/bin && \
    unzip /tmp/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    rm -fr /tmp/*

ADD rootfs /

RUN addgroup janus && \
    adduser -D -s /bin/bash -h /var/janus -g janus -G janus janus && \
    umask 0077 && \
    mkdir -p /var/janus/.ansible/tmp && \
    umask 0022 && \
    chown -R janus:janus /var/janus

COPY ./janus /usr/local/bin/

EXPOSE 8800
