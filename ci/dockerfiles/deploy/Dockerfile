FROM ruby:2.5

RUN apt-get update -y && apt-get install -y jq unzip
RUN curl -L -o /usr/local/bin/bosh https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-3.0.1-linux-amd64 && echo "ccc893bab8b219e9e4a628ed044ebca6c6de9ca0  /usr/local/bin/bosh" | shasum -c - && chmod +x /usr/local/bin/bosh
RUN curl -L "https://cli.run.pivotal.io/stable?release=linux64-binary&source=github" | tar -zx && mv cf /usr/local/bin/cf
RUN curl -L https://github.com/github/hub/releases/download/v2.2.9/hub-linux-amd64-2.2.9.tgz | tar -zx && mv hub-linux-amd64-2.2.9/bin/hub /usr/local/bin/hub

# bbl and dependencies
RUN \
  wget https://github.com/cloudfoundry/bosh-bootloader/releases/download/v6.3.0/bbl-v6.3.0_linux_x86-64 -P /tmp && \
  mv /tmp/bbl-* /usr/local/bin/bbl && \
  cd /usr/local/bin && \
  echo "8a5902bf66fe721dbb78f57c2ea4e9f04ab5b0e7  bbl" | sha1sum -c - && \
  chmod +x bbl
RUN \
  wget https://releases.hashicorp.com/terraform/0.11.1/terraform_0.11.1_linux_amd64.zip -P /tmp && \
  cd /tmp && \
  echo "fcf9e6aadc002be1905f94ecefaff2f505d4ef38  terraform_0.11.1_linux_amd64.zip" | sha1sum -c - && \
  unzip /tmp/terraform_0.11.1_linux_amd64.zip -d /tmp && \
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
