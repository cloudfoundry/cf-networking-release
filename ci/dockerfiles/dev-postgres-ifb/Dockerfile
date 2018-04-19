FROM c2cnetworking/dev-postgres
RUN apt-get update -y && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  wget \
  make \
  gcc \
  bc \
  binutils \
  sed \
  ncurses-dev \
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


ARG KERNELVER=4.9.36
RUN wget -nv -P /srv https://www.kernel.org/pub/linux/kernel/v4.x/linux-$KERNELVER.tar.gz \
 && tar -C /srv -zxf /srv/linux-$KERNELVER.tar.gz \
 && rm -f /srv/linux-$KERNELVER.tar.gz

# build kernel modules
RUN cd /srv/linux-$KERNELVER \
 && make defconfig

ADD config /srv/linux-$KERNELVER/.config

# enable modules
RUN cd /srv/linux-$KERNELVER \
 # build modules
 && make oldconfig \
 && make modules_prepare \
 && make modules \
 && make modules_install \
 && make clean

WORKDIR /srv/linux-$KERNELVER
