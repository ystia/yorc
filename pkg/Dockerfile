FROM loicalbertin/alpine-consul

# 0.7.3 last stable version
ENV TERRAFORM_VERSION=0.7.3

ADD rootfs /

RUN apk add --update ansible openssh-client && \
    rm -rf /var/cache/apk/* && \
    cd /tmp && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    cd /usr/local/bin && \
    unzip /tmp/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    addgroup janus && \
    adduser -D -s /bin/bash -h /var/janus -g janus -G janus janus && \
    umask 0077 && \
    mkdir -p /var/janus/.ansible/tmp && \
    umask 0022 && \
    chown -R janus:janus /var/janus && \
    chmod 400 /var/janus/.ssh/*.pem && \
    rm -fr /tmp/*

COPY ./janus /usr/local/bin/

EXPOSE 8800
