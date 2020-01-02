FROM golang

RUN apt-get update -y && apt-get install -y jq curl git vim
RUN curl -L "https://cli.run.pivotal.io/stable?release=linux64-binary&source=github" | tar -zx && mv cf /usr/local/bin/cf
RUN curl -L -o /usr/local/bin/bbl https://github.com/cloudfoundry/bosh-bootloader/releases/download/v6.3.0/bbl-v6.3.0_linux_x86-64 && chmod +x /usr/local/bin/bbl
RUN curl -L -o /usr/local/bin/bosh https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-3.0.1-linux-amd64 && echo "ccc893bab8b219e9e4a628ed044ebca6c6de9ca0  /usr/local/bin/bosh" | shasum -c - && chmod +x /usr/local/bin/bosh
RUN curl -L  "https://github.com/cloudfoundry-incubator/credhub-cli/releases/download/2.5.1/credhub-linux-2.5.1.tgz" | tar -zx && mv credhub /usr/local/bin

CMD /bin/bash
