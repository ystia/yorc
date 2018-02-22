FROM laurentg/alpine-consul:1.0.4_consul-1.0.6
ARG TERRAFORM_VERSION
ARG ANSIBLE_VERSION
# Update terraform default when possible
ENV TERRAFORM_VERSION ${TERRAFORM_VERSION:-0.9.8}
ENV ANSIBLE_VERSION ${ANSIBLE_VERSION:-2.4.1.0}

ADD rootfs /

# Python is required here as it should not be removed automatically when uninstalling python-dev
RUN apk add --update make openssh-client python python-dev py-pip gcc musl-dev libffi-dev openssl-dev && \
    pip install ansible==${ANSIBLE_VERSION} && \
    cd /tmp && \
    curl -O https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    cd /usr/local/bin && \
    unzip /tmp/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    addgroup yorc && \
    adduser -D -s /bin/bash -h /var/yorc -g yorc -G yorc yorc && \
    umask 0077 && \
    mkdir -p /var/yorc/.ansible/tmp && \
    umask 0022 && \
    chown -R yorc:yorc /var/yorc && \
    chmod 400 /var/yorc/.ssh/*.pem && \
    apk del make py-pip python-dev gcc musl-dev libffi-dev openssl-dev && \
    rm -rf /var/cache/apk/* && \
    rm -fr /tmp/*

COPY ./yorc /usr/local/bin/

EXPOSE 8800
