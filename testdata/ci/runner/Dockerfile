FROM ubuntu:18.04

RUN apt-get update && \
    apt-get install -y \
        build-essential \
        curl \
        gettext \
        libgtk-3-0 \
        libgtk-3-dev \
        libdbus-glib-1-2 \
        python3 \
        python3-dev \
        python3-venv \
        ssh \
        unzip \
        wget \
        zip \
    && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY rootfs /
