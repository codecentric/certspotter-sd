FROM golang:1.15 as build

COPY . /usr/share/repo
WORKDIR /usr/share/repo

RUN apt-get update && apt-get install -y \
    ca-certificates
RUN make

FROM debian:stable
LABEL maintainer="Felix Ehrenpfort <felix.ehrenpfort@codecentric.cloud>"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/repo/out/certspotter-sd /usr/local/bin/certspotter-sd
COPY example/certspotter-sd.yml /etc/prometheus/certspotter-sd.yml

RUN mkdir -p /var/lib/certspotter-sd && \
    chown -R nobody:nogroup etc/prometheus /var/lib/certspotter-sd

USER       nobody
VOLUME     [ "/var/lib/certspotter-sd" ]
ENTRYPOINT [ "/usr/local/bin/certspotter-sd" ]
CMD        [ "--config.file=/etc/prometheus/certspotter-sd.yml" ]

