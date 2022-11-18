FROM ystia/alpine-consul:1.1.0_consul-1.11.3
ARG TERRAFORM_VERSION
ARG ANSIBLE_VERSION
ARG TF_CONSUL_PLUGIN_VERSION
ARG TF_AWS_PLUGIN_VERSION
ARG TF_GOOGLE_PLUGIN_VERSION
ARG TF_OPENSTACK_PLUGIN_VERSION
# Update terraform default when possible
ENV TERRAFORM_VERSION ${TERRAFORM_VERSION:-0.11.8}
ENV ANSIBLE_VERSION ${ANSIBLE_VERSION:-2.10.0}
ENV TF_CONSUL_PLUGIN_VERSION ${TF_CONSUL_PLUGIN_VERSION:-2.1.0}
ENV TF_AWS_PLUGIN_VERSION ${TF_AWS_PLUGIN_VERSION:-1.36.0}
ENV TF_GOOGLE_PLUGIN_VERSION ${TF_GOOGLE_PLUGIN_VERSION:-1.18.0}
ENV TF_OPENSTACK_PLUGIN_VERSION ${TF_OPENSTACK_PLUGIN_VERSION:-1.9.0}
ENV YORC_TERRAFORM_PLUGINS_DIR /var/terraform/plugins

ADD rootfs /

# Python is required here as it should not be removed automatically when uninstalling python-dev
# We do not install the whole docker package but just docker-cli

RUN apk add --update make openssh-client python3 python3-dev gcc musl-dev libffi-dev openssl-dev docker-cli cargo && \
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs --output /tmp/rustup.sh && \
    chmod +x /tmp/rustup.sh && /tmp/rustup.sh -y && source "$HOME/.cargo/env" && \
    python3 -m ensurepip --upgrade && \
    if [ ! -e /usr/bin/pip ]; then ln -s pip3 /usr/bin/pip ; fi && \
    if [ ! -e /usr/bin/python ]; then ln -sf /usr/bin/python3 /usr/bin/python; fi && \
    pip install ansible==${ANSIBLE_VERSION} docker-py netaddr jmespath && \
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
    echo "Copy Terraform providers plugins" && \
    cd ${YORC_TERRAFORM_PLUGINS_DIR} && \
    curl -O https://releases.hashicorp.com/terraform-provider-consul/${TF_CONSUL_PLUGIN_VERSION}/terraform-provider-consul_${TF_CONSUL_PLUGIN_VERSION}_linux_amd64.zip && \
    unzip terraform-provider-consul_${TF_CONSUL_PLUGIN_VERSION}_linux_amd64.zip && \
    curl -O https://releases.hashicorp.com/terraform-provider-null/1.0.0/terraform-provider-null_1.0.0_linux_amd64.zip && \
    unzip terraform-provider-null_1.0.0_linux_amd64.zip && \
    curl -O https://releases.hashicorp.com/terraform-provider-aws/${TF_AWS_PLUGIN_VERSION}/terraform-provider-aws_${TF_AWS_PLUGIN_VERSION}_linux_amd64.zip && \
    unzip terraform-provider-aws_${TF_AWS_PLUGIN_VERSION}_linux_amd64.zip && \
    curl -O https://releases.hashicorp.com/terraform-provider-google/${TF_GOOGLE_PLUGIN_VERSION}/terraform-provider-google_${TF_GOOGLE_PLUGIN_VERSION}_linux_amd64.zip && \
    unzip terraform-provider-google_${TF_GOOGLE_PLUGIN_VERSION}_linux_amd64.zip && \
    curl -O https://releases.hashicorp.com/terraform-provider-openstack/${TF_OPENSTACK_PLUGIN_VERSION}/terraform-provider-openstack_${TF_OPENSTACK_PLUGIN_VERSION}_linux_amd64.zip && \
    unzip terraform-provider-openstack_${TF_OPENSTACK_PLUGIN_VERSION}_linux_amd64.zip && \
    chmod 775 ${YORC_TERRAFORM_PLUGINS_DIR}/* && \
    echo "Cleaning up" && \
    apk del make py-pip python3-dev gcc musl-dev libffi-dev openssl-dev && \
    rustup self uninstall -y && \
    rm -rf /var/cache/apk/* && \
    rm -fr /tmp/*

COPY ./yorc /usr/local/bin/

EXPOSE 8800

ARG BUILD_DATE
ARG VCS_REF
ARG YORC_VERSION

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.vcs-url="https://github.com/ystia/yorc.git" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.schema-version="1.0.0-rc1" \
      org.label-schema.vendor="Bull Atos Technologies" \
      org.label-schema.version=${YORC_VERSION}
