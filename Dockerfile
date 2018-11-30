FROM ubuntu:xenial
LABEL maintainer=https://github.com/cloudfoundry-incubator/pxc-release

# Install base packages needed
RUN apt-get update && \
  apt-get -y install \
    lsb-release \
    software-properties-common \
    wget \
    vim.tiny \
    git

# Install golang 1.11
RUN add-apt-repository -y ppa:longsleep/golang-backports && \
  apt-get update && \
  apt-get install -y golang-go

# Get apt repos for pxc
# Set a root user password for pxc when it boots up, and then install it. This sets the password to be "root"
RUN wget https://repo.percona.com/apt/percona-release_0.1-6.$(lsb_release -sc)_all.deb \
  && dpkg -i percona-release_0.1-6.$(lsb_release -sc)_all.deb

RUN  echo "percona-xtradb-cluster-server-5.7 percona-xtradb-cluster-server-5.7/root-pass password root" | debconf-set-selections \
  && echo "percona-xtradb-cluster-server-5.7 percona-xtradb-cluster-server-5.7/re-root-pass password root" | debconf-set-selections

# this may not be needed now that the echo debselections work
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
  apt-get -y install --fix-missing \
    percona-xtradb-cluster-full-57

RUN mkdir -p /go/src/github.com/cloudfoundry/galera-init
RUN chown -R mysql:mysql /go

USER mysql
ENV GOPATH=/go
ENV PATH=$PATH:$GOPATH/bin

RUN go get github.com/onsi/ginkgo/ginkgo && go get github.com/onsi/gomega/...
RUN go get github.com/cloudfoundry-incubator/galera-healthcheck && go get github.com/maxbrunsfeld/counterfeiter
WORKDIR /go/src/github.com/cloudfoundry/galera-init

USER root
COPY . /go/src/github.com/cloudfoundry/galera-init
RUN chown -R mysql:mysql /go/src/github.com/cloudfoundry/galera-init /var/lib/galera /var/lib/mysql* /var/run/mysql*
COPY ./integration_test/fixtures/abraham/mylogin.cnf /var/vcap/jobs/pxc-mysql/config/mylogin.cnf
RUN echo "init_file = /go/src/github.com/cloudfoundry/galera-init/integration_test/fixtures/abraham/db_init" >> /etc/mysql/percona-xtradb-cluster.conf.d/mysqld.cnf

RUN mkdir -p /var/vcap/jobs/pxc-mysql/config \
  /var/vcap/data/pxc-mysql/files \
  /var/vcap/packages/pxc-utils /var/vcap/sys \
  /var/vcap/jobs/pxc-mysql/config/ \
  /var/vcap/jobs/pxc-mysql \
  /var/vcap/sys/log/pxc-mysql \
  /var/vcap/sys/run/pxc-ctl \
  /var/vcap/sys/run/pxc-mysql \
  && ln -s /etc/mysql/my.cnf /var/vcap/jobs/pxc-mysql/config/my.cnf \
  && chown -R mysql:mysql /var/vcap



USER mysql

RUN go get github.com/pkg/errors

RUN find /var/lib/mysql -type f -exec touch {} \;

CMD find /var/lib/mysql -type f -exec touch {} \; && ./bin/test-integration
