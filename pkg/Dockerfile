FROM loicalbertin/alpine-consul:1.0.3_consul-0.8.1
ARG TERRAFORM_VERSION
ARG ANSIBLE_VERSION
# Update terraform default when possible
ENV TERRAFORM_VERSION ${TERRAFORM_VERSION:-0.9.8}
ENV ANSIBLE_VERSION ${ANSIBLE_VERSION:-2.3.1.0}

ADD rootfs /

# Python is required here as it should not be removed automatically when uninstalling python-dev
RUN apk add --update make openssh-client python python-dev py-pip gcc musl-dev libffi-dev openssl-dev && \
    pip install ansible==${ANSIBLE_VERSION} && \
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
    apk del make py-pip python-dev gcc musl-dev libffi-dev openssl-dev && \
    rm -rf /var/cache/apk/* && \
    rm -fr /tmp/*

COPY ./janus /usr/local/bin/

EXPOSE 8800
