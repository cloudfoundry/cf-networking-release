#!/bin/bash

set -e -u

cd netman-release
export GOPATH=$PWD

declare -a packages=(
  "src/lib"
  "src/policy-server"
  "src/github.com/cloudfoundry-incubator/guardian-cni-adapter"
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

if [ ${NO_POSTGRES:-"false"} = "true" ]; then
  echo "skipping postgres"
else
  bootPostgres
fi


for dir in "${packages[@]}"; do
  pushd $dir
    extraFlags="${GINKGO_EXTRA_FLAGS:-""}"
    ginkgo -r -p -randomizeAllSpecs -randomizeSuites $extraFlags "$@"
  popd
done
