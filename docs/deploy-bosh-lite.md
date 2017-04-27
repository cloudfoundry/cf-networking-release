# Deploy to bosh-lite

## Option 1: Using `cf-deployment`

- Option 1: use the script
  ```bash
  pushd ~/workspace/cf-networking-release
    ./scripts/deploy-to-bosh-lite
  popd
  ```

- Option 2: deploy by hand
Follow the instructions [here](https://github.com/cloudfoundry/bosh-deployment/blob/master/docs/bosh-lite-on-vbox.md) to install `bosh-lite` using `BOSH CLI v2` on your machine.

This deployment option uses the new tooling:
- the new Golang [bosh-cli](https://github.com/cloudfoundry/bosh-cli)
- [bosh-deployment](https://github.com/cloudfoundry/bosh-deployment)
- [cf-deployment](https://github.com/cloudfoundry/cf-deployment), refer to our [release notes](https://github.com/cloudfoundry-incubator/cf-networking-release/releases) to get information on validated versions

It assumes you have a BOSH director on Virtualbox that was created using `bosh create-env`.

You should have a private directory in which you hold the `creds.yml` file for your bosh director

```bash
cd ~/deployments/vbox
ls
# creds.yml
```

```bash
export BOSH_CA_CERT=$(bosh int ~/deployments/vbox/creds.yml --path /director_ssl/ca)
export BOSH_CLIENT=admin
export BOSH_CLIENT_SECRET=$(bosh int ~/deployments/vbox/creds.yml --path /admin_password)
export BOSH_ENVIRONMENT=vbox
export BOSH_DEPLOYMENT=cf
```

We need to enable `br_netfilter` module on the bosh-lite VM.

```bash
umask 077; touch ~/deployments/vbox/director_priv.key
bosh int ~/deployments/vbox/creds.yml --path /jumpbox_ssh/private_key > ~/deployments/vbox/director_priv.key
ssh jumpbox@192.168.50.6 -i ~/deployments/vbox/director_priv.key 'sudo modprobe br_netfilter && lsmod | grep br_netfilter'
```

If you are upgrading an existing `cf-deployment`, this same directory should hold your `deployment-vars.yml`
file containing credentials for your existing deployment.

Set the `cf-deployment` cloud-config:
```
bosh -e vbox update-cloud-config ~/workspace/cf-deployment/bosh-lite/cloud-config.yml
```

Upload `cf-networking-release`, e.g.
```
bosh upload-release https://bosh.io/d/github.com/cloudfoundry-incubator/cf-networking-release
```

Then deploy
```bash
bosh deploy ~/workspace/cf-deployment/cf-deployment.yml \
  -o ~/workspace/cf-networking-release/manifest-generation/opsfiles/cf-networking.yml \
  -o ~/workspace/cf-deployment/operations/bosh-lite.yml \
  -o ~/workspace/cf-networking-release/manifest-generation/opsfiles/postgres.yml \
  --vars-store ~/deployments/vbox/deployment-vars.yml \
  -v system_domain=bosh-lite.com
```

## Option 2 (deprecated): Using `cf-release` with `diego-release` tooling

Follow the instructions [here](https://github.com/cloudfoundry/bosh-lite) to install `bosh-lite` on your machine.

Ensure that `br_netfilter` is enabled on your vagrant box:
```bash
pushd ~/workspace/bosh-lite
  vagrant ssh -c 'sudo modprobe br_netfilter'
popd
```
or edit your `Vagrantfile` to include
```ruby
config.vm.provision "shell", inline: "sudo modprobe br_netfilter"
```

Upload the latest `bosh-lite` stemcell 
```bash
bosh upload stemcell https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
```

Or download the stemcell and manually upload it to `bosh-lite` (potentially faster)
```bash
curl -L -o bosh-lite-stemcell-latest.tgz https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent
bosh upload stemcell bosh-lite-stemcell-latest.tgz
```

Then grab the required releases
```bash
pushd ~/workspace
  git clone https://github.com/cloudfoundry/diego-release
  git clone https://github.com/cloudfoundry/cf-release
  git clone https://github.com/cloudfoundry-incubator/cf-networking-release
popd
```

Deploy:
```
bosh upload release https://bosh.io/d/github.com/cloudfoundry/cf-release
bosh upload release https://bosh.io/d/github.com/cloudfoundry/diego-release
bosh upload release https://bosh.io/d/github.com/cloudfoundry/garden-runc-release
bosh upload release https://bosh.io/d/github.com/cloudfoundry/cflinuxfs2-rootfs-release
bosh upload release https://bosh.io/d/github.com/cloudfoundry-incubator/cf-networking-release
```

Then
```bash
pushd ~/workspace/cf-networking-release
  bosh upload release releases/cf-networking-<LATEST-VERSION>.yml

  ./scripts/generate-bosh-lite-manifests
  bosh -d bosh-lite/deployments/cf_networking.yml deploy
  bosh -d bosh-lite/deployments/diego_cf_networking.yml deploy
popd
```


# Kicking the tires

Try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.


## Known issues with bosh-lite
There is a known issue where VMs on `bosh-lite` can start failing,
particularly if the host machine goes to sleep.

If you run `bosh vms` and see any failing VMs, then you can either recreate the
individual failing vm(s) with
```
bosh recreate <vm_name>
```
or you can run
```
bosh deploy --recreate
```
to recreate all VMs.

# Syslog forwarding
This is not specific to CF Networking, but is useful for debugging during development.

To forward all logs from your bosh-lite to a syslog destination (like Papertrail),
add the following block to `manifest-generation/stubs/bosh-lite-cf.yml`:
```yaml
  syslog_daemon_config:
    address: some-syslog-host.example.com
    port: 12345
    transport: udp
```
