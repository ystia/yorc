FROM buildpack-deps:bionic-scm

ENV GOLANG_VERSION=1.13.6 GOLANG_TGZ_CHECKSUM=a1bc06deb070155c4f67c579f896a45eeda5a8fa54f35ba233304074c4abbbbd

RUN apt-get update && \
    apt-get install -y \
        build-essential \
        curl \
        fonts-liberation \
        gettext \
        libappindicator3-1 \
        libasound2 \
        libgtk-3-0 \
        libgtk-3-dev \
        libdbus-glib-1-2 \
        libnspr4 \
        libnss3 \
        libxss1 \
        python3 \
        python3-dev \
        python3-venv \
        ssh \
        unzip \
        wget \
        xdg-utils \
        zip \
    && \
    wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb  -O /tmp/google-chrome-stable_current_amd64.deb && \
    dpkg -i /tmp/google-chrome-stable_current_amd64.deb && \
    rm /tmp/google-chrome-stable_current_amd64.deb && \
    wget -O go.tgz "https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz" && \
	echo "${GOLANG_TGZ_CHECKSUM} *go.tgz" | sha256sum -c - && \
	tar -C /usr/local -xzf go.tgz && \
	rm go.tgz && \
    mkdir -p /go/src /go/bin && \
    export GOPATH=/go && \
    export PATH="$GOPATH/bin:/usr/local/go/bin:$PATH" && \
    go get github.com/DATA-DOG/godog/cmd/godog && \
    chmod -R 777 /go && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY rootfs /

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
