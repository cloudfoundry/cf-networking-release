To use:

1. Clone the [prometheus-boshrelease](https://github.com/bosh-prometheus/prometheus-boshrelease) to ~/workspace
1. Deploy your CF with the opsfile [add-prometheus-uaa-clients.yml](https://github.com/bosh-prometheus/prometheus-boshrelease/blob/master/manifests/operators/cf/add-prometheus-uaa-clients.yml)
1. Look up the values in credhub for the created clients:
    1. `credhub find -n uaa_clients_cf_exporter_secret`
    1. `credhub get -n <path from above>`
    1.  repeat for `uaa_clients_firehose_exporter_secret`
1. Fill empty values in `./change-me-deployment-vars.yml` (Note that bosh client/secret doesn't work for username/password)
1. Run `./deploy.sh` to deploy prometheus and grafana that scrapes from your cf/bosh
