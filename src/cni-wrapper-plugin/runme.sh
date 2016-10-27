#! /bin/bash

set -e -u
set -o pipefail

echo '{ "ReportResult": "{}" }'> /tmp/noop

export CNI_COMMAND=ADD
export CNI_CONTAINERID=some-container-id
export CNI_ARGS=DEBUG=/tmp/noop
export CNI_NETNS=/some/netns/path
export CNI_IFNAME=some-eth0
export CNI_PATH=${PWD}

#INPUT_NOOP=$(cat <<END
#{"delegate":{"cniVersion":"0.2.0","some":"stdin-json"},"name":"cni-noop","type":"noop"}
#END
#)
#
#echo  $INPUT_NOOP | jq .
#echo  $INPUT_NOOP | ./noop
#
#exit 
#INPUT_NOOP=$(cat <<END
#{
#  "name": "cni-noop",
#  "type": "noop",
#  "delegate":
#  {"some":"stdin-json", "cniVersion": "0.2.0"}
#}
#END
#)
#
#
#echo  $INPUT_NOOP | jq .
#echo  $INPUT_NOOP | ./noop
#exit 0
#
go build
INPUT_WRAPPER=$(cat <<END
{
  "name": "cni-wrapper",
  "type": "wrapper",
	"cniVersion": "0.2.0",
  "datastore": "/path/to/datastore",
	"delegate": {
			"name": "cni-noop",
			"type": "noop",
      "delegate":
        {"some":"stdin-json", "cniVersion": "0.2.0"}
   }
}
END
)

echo  $INPUT_WRAPPER | jq .
echo  $INPUT_WRAPPER | ./cni-wrapper-plugin
