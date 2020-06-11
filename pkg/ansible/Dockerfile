# Dockerfile for building the sandbox container

FROM alpine:3.10

ENV ANSIBLE_VERSION=2.7.9

RUN apk add --no-cache --progress bash python3 openssl ca-certificates openssh sshpass && \
    addgroup -g 1000 ansible && \
    adduser -D -s /bin/bash -g ansible -G ansible -u 1000 ansible && \
    if [ ! -e /usr/bin/python ]; then ln -sf /usr/bin/python3 /usr/bin/python; fi && \
	apk --update add --virtual build-dependencies python3-dev libffi-dev openssl-dev build-base && \
    pip3 install --upgrade pip && \
	pip3 install ansible==${ANSIBLE_VERSION} netaddr jmespath && \
	apk del build-dependencies && \
	rm -rf /var/cache/apk/*