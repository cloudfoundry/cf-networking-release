FROM ruby

RUN apt-get update -y && apt-get install -y jq
RUN curl -L -o /usr/local/bin/bosh https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-2.0.17-linux-amd64 && chmod +x /usr/local/bin/bosh
RUN curl -L "https://cli.run.pivotal.io/stable?release=linux64-binary&source=github" | tar -zx && mv cf /usr/local/bin/cf
RUN curl -L https://github.com/github/hub/releases/download/v2.2.9/hub-linux-amd64-2.2.9.tgz | tar -zx && mv hub-linux-amd64-2.2.9/bin/hub /usr/local/bin/hub

CMD /bin/bash
