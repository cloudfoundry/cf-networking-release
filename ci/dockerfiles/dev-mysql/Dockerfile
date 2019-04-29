FROM mysql:5.7

RUN apt-get update -y && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  dnsutils \
  git \
  jq \
  iptables \
  g++ \
  netcat \
  kmod \
  iproute2 \
  iputils-ping \
  &&  apt-get autoremove -yqq \
  &&  apt-get clean \
  &&  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN curl -L "https://cli.run.pivotal.io/stable?release=linux64-binary&source=github" | tar -zx && mv cf /usr/local/bin/cf

ADD go*.tar.gz /usr/local

ENV GOPATH /gopath
ENV GOBIN /gopath/bin
ENV PATH $PATH:/usr/local/go/bin:$GOPATH/bin

RUN go get github.com/tools/godep
RUN go get github.com/onsi/ginkgo

RUN go install github.com/onsi/ginkgo/ginkgo

CMD /bin/bash
