#!/bin/bash

set -euo pipefail
set -x

datacenterID=$(ionosctl datacenter list -ojson | jq -r '.Resources[] | select(.properties.name == "joe-test") | .id')

# ionosctl image list | grep "Image\|ubuntu-22"

for i in {1..3}; do
  export serverName="controlplane-$i"
  export volName="vol-$serverName"
  ionosctl volume create --datacenter-id "$datacenterID" --name "$volName" --image-alias ubuntu:22.04 --size 80 --type SSD --ssh-key-paths /home/jmanser/.ssh/id_rsa.pub
  ionosctl server create -n "$serverName" --datacenter-id "$datacenterID" --cores 4 --ram 4096 --cpu-family INTEL_SKYLAKE
  serverID=""
  while [ -z "$serverID" ]; do
    serverID=$(ionosctl server list --datacenter-id "$datacenterID" -ojson | jq -r '.Resources[] | select(.properties.name == env.serverName ) | .id')
    sleep 5
  done
  echo "server id is: $serverID"
  volID=""
  while [ -z "$volID" ]; do
    volID=$(ionosctl volume list --datacenter-id "$datacenterID" -ojson | jq -r '.Resources[] | select(.properties.name == env.volName ) | .id')
    sleep 5
  done
  ionosctl server volume attach --datacenter-id "$datacenterID" --volume-id "$volID" --server-id "$serverID"

  ionosctl nic create --datacenter-id "$datacenterID" --server-id "$serverID"
done

for i in {1..1}; do
  export serverName="worker-$i"
  export volName="vol-$serverName"
  ionosctl volume create --datacenter-id "$datacenterID" --name "$volName" --image-alias ubuntu:22.04 --size 80 --type SSD --ssh-key-paths /home/jmanser/.ssh/id_rsa.pub
  ionosctl server create -n "$serverName" --datacenter-id "$datacenterID" --cores 4 --ram 4096 --cpu-family INTEL_SKYLAKE
  serverID=""
  while [ -z "$serverID" ]; do
    serverID=$(ionosctl server list --datacenter-id "$datacenterID" -ojson | jq -r '.Resources[] | select(.properties.name == env.serverName ) | .id')
    sleep 5
  done
  echo "server id is: $serverID"
  volID=""
  while [ -z "$volID" ]; do
    volID=$(ionosctl volume list --datacenter-id "$datacenterID" -ojson | jq -r '.Resources[] | select(.properties.name == env.volName ) | .id')
    sleep 5
  done
  ionosctl server volume attach --datacenter-id "$datacenterID" --volume-id "$volID" --server-id "$serverID"

  ionosctl nic create --datacenter-id "$datacenterID" --server-id "$serverID"
done

## use below to get server IP (it works only in the simplest case of only 1 nic and IP)
# ionosctl nic list --datacenter-id "$datacenterID" --server-id $serverID  -ojson | jq '.Resources[0].properties.ips[0]'
