#!/bin/bash

set -e -u

cd netman-release
export GOPATH=$PWD

declare -a packages=(
  "src/cf-pusher"
  "src/cli-plugin"
  "src/cni-wrapper-plugin"
  "src/example-apps"
  "src/flannel-watchdog"
  "src/lib"
  "src/netmon"
  "src/policy-server"
  )

declare -a serial_packages=(
  "src/garden-external-networker"
  "src/vxlan-policy-agent"
  )

function bootPostgres {
	echo -n "booting postgres"
	(/docker-entrypoint.sh postgres &> /var/log/postgres-boot.log) &
	trycount=0
	for i in `seq 1 9`; do
		set +e
		psql -h localhost -U postgres -c '\conninfo' &>/dev/null
		exitcode=$?
		set -e
		if [ $exitcode -eq 0 ]; then
			echo "connection established to postgres"
			return 0
		fi
		echo -n "."
		sleep 1
	done
	echo "unable to connect to postgres"
	exit 1
}

function bootMysql {
	echo -n "booting mysql"
	(MYSQL_ROOT_PASSWORD=password  /entrypoint.sh mysqld &> /var/log/postgres-boot.log) &
	trycount=0
	for i in `seq 1 30`; do
		set +e
		echo '\s;' | mysql -h 127.0.0.1 -u root --password="password" &>/dev/null
		exitcode=$?
		set -e
		if [ $exitcode -eq 0 ]; then
			echo "connection established to mysql"
			return 0
		fi
		echo -n "."
		sleep 1
	done
	echo "unable to connect to mysql"
	exit 1
}

if [ ${NO_DB:-"false"} = "true" ]; then
  echo "skipping database"
else
  if [ ${MYSQL:-"false"} = "true" ]; then
    bootMysql
  else
    bootPostgres
  fi
fi

if [ "${1:-""}" = "" ]; then
  for dir in "${packages[@]}"; do
    pushd $dir
      ginkgo -r -p --race -randomizeAllSpecs -randomizeSuites "${@:2}"
    popd
  done
  for dir in "${serial_packages[@]}"; do
    pushd $dir
      ginkgo -r -randomizeAllSpecs -randomizeSuites "${@:2}"
    popd
  done
else
  testdir="$1"
  pushd $testdir
    ginkgo -r -randomizeAllSpecs -randomizeSuites "${@:2}"
  popd
fi
