FROM loicalbertin/alpine-consul

# 0.6.16 is dynamically linked which causes problems (wait for 0.6.17 or 0.7.0)
ENV TERRAFORM_VERSION=0.6.15

RUN apk add --update ansible openssh-client && \
    rm -rf /var/cache/apk/*

ADD rootfs /

RUN cd /tmp && \
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
