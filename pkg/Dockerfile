FROM starlings/janus:59

ENV  http_proxy http://193.56.47.20:8080
ENV  https_proxy http://193.56.47.20:8080
RUN apk update
RUN apk add openssh

COPY terraform-provider-slurm /usr/local/bin/
COPY scripts /usr/local/bin/scripts

COPY janus /usr/local/bin/

COPY .terraformrc /var/janus/

EXPOSE 8800
