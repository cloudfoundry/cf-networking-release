FROM ruby

RUN apt-get update -y && apt-get install -y jq unzip
RUN curl -L -o /usr/local/bin/bosh https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-2.0.40-linux-amd64 && shasum /usr/local/bin/bosh | grep '0045668aa6582ddd518cfb2b6134eba439bbb94d' && chmod +x /usr/local/bin/bosh
RUN curl -L "https://cli.run.pivotal.io/stable?release=linux64-binary&source=github" | tar -zx && mv cf /usr/local/bin/cf
RUN curl -L https://github.com/github/hub/releases/download/v2.2.9/hub-linux-amd64-2.2.9.tgz | tar -zx && mv hub-linux-amd64-2.2.9/bin/hub /usr/local/bin/hub

# bbl and dependencies
RUN \
  wget https://github.com/cloudfoundry/bosh-bootloader/releases/download/v3.2.6/bbl-v3.2.6_linux_x86-64 -P /tmp && \
  mv /tmp/bbl-* /usr/local/bin/bbl && \
  cd /usr/local/bin && \
  echo "0393afe06f5ca1664553895f41c0f3d6186f191b bbl" | sha1sum -c - && \
  chmod +x bbl
RUN \
  wget https://releases.hashicorp.com/terraform/0.9.7/terraform_0.9.7_linux_amd64.zip -P /tmp && \
  cd /tmp && \
  echo "f7d9d2110acccb5b24fec86f2651f3f07af493c9 terraform_0.9.7_linux_amd64.zip" | sha1sum -c - && \
  unzip /tmp/terraform_0.9.7_linux_amd64.zip -d /tmp && \
  mv /tmp/terraform /usr/local/bin/terraform && \
  cd /usr/local/bin && \
  chmod +x terraform && \
  rm -rf /tmp/*
RUN \
  wget https://s3.amazonaws.com/bosh-init-artifacts/bosh-init-0.0.99-linux-amd64 -P /tmp && \
  mv /tmp/bosh-init-0.0.99-linux-amd64 /usr/local/bin/bosh-init && \
  cd /usr/local/bin && \
  echo "00ccaf07a11bd8206407f83f1e606e16f3475bf3 bosh-init" | sha1sum -c - && \
  chmod +x bosh-init

CMD /bin/bash
