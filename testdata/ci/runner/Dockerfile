FROM ubuntu:18.04

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
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY rootfs /
