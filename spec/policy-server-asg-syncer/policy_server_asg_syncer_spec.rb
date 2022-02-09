require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'policy-server-asg-syncer job template rendering' do
    let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
    let(:release) {ReleaseDir.new(release_path)}
    let(:job) {release.job('policy-server-asg-syncer')}
    let(:cc_hostname) {'some-cc-hostname'}
    let(:cc_port) {4567}
    let(:merged_manifest_properties) do
      {
        'disable' => false,
        'uaa_client' => 'some-uaa-client',
        'uaa_client_secret' => 'some-uaa-client-secret',
        'uaa_ca' => 'some-uaa-ca',
        'uaa_hostname' => 'some-uaa-hostname',
        'uaa_port' => 3456,
        'cc_hostname' => cc_hostname,
        'cc_port' => cc_port,
        'log_level' => 'debug',
        'database' => {
          'connect_timeout_seconds' => 30,
        },
        'locket' => {
          'ca_cert' => 'the locket ca cert',
          'client_cert' => 'the locket cert',
          'client_key' => 'the locket key',
        }
      }
    end

    let(:dbconn_host) {'some-database-host'}

    let(:db_properties) do
      {
        'name' => 'some-database-name',
        'type' => 'some-database-type',
        'username' => 'some-database-username',
        'password' => 'some-database-password',
        'host' => dbconn_host,
        'require_ssl' => true,
        'port' => 4321,
        'ca_cert' => 'some ca cert',
        'skip_hostname_validation' => true,
      }
    end

    let(:dbconn_link) do
      Link.new(
        name: 'dbconn',
        instances: [LinkInstance.new()],
        properties: {
          'database' => db_properties
        }
      )
    end

    let(:db_link) do
      Link.new(name: 'database', instances: [LinkInstance.new(address: 'some-other-database-address')])
    end

    let(:cc_link) do
      Link.new(
        name: 'cloud_controller_https_endpoint',
        properties: {
          'cc' => {
            'internal_service_hostname' => 'cc.service.internal',
            'public_tls' => {
              'port' => '443',
              'ca_cert' => 'the-cc-ca-cert'
            }
          }
        }
      )
    end

    let(:links) {[dbconn_link, db_link]}

    describe 'database_ca.crt' do
      let(:template) {job.template('config/certs/database_ca.crt')}
      it 'renders the cert' do
        cert = template.render(merged_manifest_properties, consumes: links)
        expect(cert).to eq('some ca cert')
      end
    end

    describe 'policy-server-asg-syncer.json' do
      let(:template) {job.template('config/policy-server-asg-syncer.json')}

      it 'creates a config/policy-server-asg-syncer.json from properties' do
        config = JSON.parse(template.render(merged_manifest_properties, consumes: links))
        expect(config).to eq({
          'uuid' => 'xxxxxx-xxxxxxxx-xxxxx',
          'uaa_client' => 'some-uaa-client',
          'uaa_client_secret' => 'some-uaa-client-secret',
          'uaa_url' => 'https://some-uaa-hostname',
          'uaa_port' => 3456,
          'cc_ca_cert' => '/var/vcap/jobs/policy-server-asg-syncer/config/certs/cc_ca.crt',
          'cc_url' => 'http://some-cc-hostname:4567',
          'database' => {
            'user' => 'some-database-username',
            'type' => 'some-database-type',
            'password' => 'some-database-password',
            'port' => 4321,
            'database_name' => 'some-database-name',
            'host' => 'some-database-host',
            'timeout' => 30,
            'require_ssl' => true,
            'ca_cert' => '/var/vcap/jobs/policy-server-asg-syncer/config/certs/database_ca.crt',
            'skip_hostname_validation' => true,
          },
          'log_level' => 'debug',
          'log_prefix' => 'cfnetworking',
          'metron_address' => '127.0.0.1:3457',
          'skip_ssl_validation' => false,
          'uaa_ca' => '/var/vcap/jobs/policy-server-asg-syncer/config/certs/uaa_ca.crt',
          'asg_poll_interval_seconds' => 60,
          'locket_address' => 'locket.service.cf.internal:8891',
          'locket_ca_cert_file' => '/var/vcap/jobs/policy-server-asg-syncer/config/certs/locket_ca.crt',
          'locket_client_cert_file' => '/var/vcap/jobs/policy-server-asg-syncer/config/certs/locket.crt',
          'locket_client_key_file' => '/var/vcap/jobs/policy-server-asg-syncer/config/certs/locket.key',
        })
      end

      context 'when capi provides a link to the https endpoint' do
        let(:links) {[dbconn_link, db_link, cc_link]}

        before do
          merged_manifest_properties.delete('cc_hostname')
          merged_manifest_properties.delete('cc_port')
        end

        it 'uses the values from the cloud controller link' do
          policyServerJSON = JSON.parse(template.render(merged_manifest_properties, consumes: links))

          expect(policyServerJSON['cc_url']).to eq 'https://cc.service.internal:443'
          expect(policyServerJSON['cc_ca_cert']).to eq '/var/vcap/jobs/policy-server-asg-syncer/config/certs/cc_ca.crt'
        end

        describe 'cc_ca.crt' do
          let(:template) {job.template('config/certs/cc_ca.crt')}
          it 'writes the content of cc ca cert' do
            cc_ca_cert = template.render(merged_manifest_properties, consumes: links)
            expect(cc_ca_cert.strip).to eq('the-cc-ca-cert')
          end
        end
      end

      context 'when cc_hostname and cc_port property values are provided, and the link is provided' do
        let(:cc_hostname) {'use.me.pls'}
        let(:cc_port) {1234}
        let(:links) {[dbconn_link, db_link, cc_link]}

        it 'uses the property values, so the link can be overridden' do
          policyServerJSON = JSON.parse(template.render(merged_manifest_properties, consumes: links))

          expect(policyServerJSON['cc_url']).to eq 'http://use.me.pls:1234'
          expect(policyServerJSON['cc_ca_cert']).to eq '/var/vcap/jobs/policy-server-asg-syncer/config/certs/cc_ca.crt'
        end
      end

      context 'when neither the cc_hostname nor cc_port property values are provided nor is the link provided' do
        before do
          merged_manifest_properties.delete('cc_hostname')
          merged_manifest_properties.delete('cc_port')
        end

        it 'should raise an error' do
          expect {
            JSON.parse(template.render(merged_manifest_properties, consumes: links))
          }.to raise_error '`cc_hostname` and `cc_port` properties were not supplied as manifest properties, nor were found in `cloud_controller_https_endpoint` link'
        end
      end

      context 'when dbconn does not have host' do
        let(:dbconn_host) {nil}

        it 'uses database link' do
          config = JSON.parse(template.render(merged_manifest_properties, consumes: links))
          expect(config['database']['host']).to eq('some-other-database-address')
        end

        context 'when database link does not exit' do
          let(:links) {[dbconn_link]}
          it 'raises a helpful error message' do
            expect {
              JSON.parse(template.render(merged_manifest_properties, consumes: links))
            }.to raise_error('must provide dbconn link or database link')
          end
        end
      end

      it 'raises an error when the asg_poll_interval_seconds is invalid' do
        intervals = [
          'notanumber',
          0,
          -1,
          1.3,
          0.5,
          true,
          -0,
          '1',
          '0',
        ]
        intervals.each do |interval|
          merged_manifest_properties['asg_poll_interval_seconds'] = interval
          expect {
            JSON.parse(template.render(merged_manifest_properties, consumes: links))
          }.to raise_error('asg_poll_interval_seconds must be an integer greater than 0')
        end
      end

      it 'raises an error when there is no locket.address defined' do
        addrs = [
          '',
          'my-site-without-port.com',
          'http://asdf.com',
          'http://asdf:1234',
          'asdf.com:badport',
          'me+you:1234',
        ]

        addrs.each do |addr|
          merged_manifest_properties['locket']['address'] = addr
          expect {
            JSON.parse(template.render(merged_manifest_properties, consumes: links))
          }.to raise_error('the locket.address is invalid')
        end
      end

      it 'allows common domain name/ip addr combos for locket.address' do
        addrs = [
          'test.com:1234',
          '10.10.10.10:1234',
          'my-cool-site.com:1234',
        ]

        addrs.each do |addr|
          merged_manifest_properties['locket']['address'] = addr
          expect {
          JSON.parse(template.render(merged_manifest_properties, consumes: links))
        }.to_not raise_error
        end
      end
    end

    describe 'locket.ca.crt' do
      let(:template) {job.template('config/certs/locket_ca.crt')}
      describe 'When the property exits' do
        it 'renders the locket cert' do
          cert = template.render(merged_manifest_properties)
          expect(cert.strip).to eq('the locket ca cert')
        end
      end

      describe 'when the property doesn\'t exist' do
        before do
          merged_manifest_properties['locket'].delete('ca_cert')
        end

        it 'raises an error when there is no locket_ca_cert defined' do
          expect {
            template.render(merged_manifest_properties)
          }.to raise_error Bosh::Template::UnknownProperty
        end
      end
    end

    describe 'locket.crt' do
      let(:template) {job.template('config/certs/locket.crt')}
      describe 'When the property exits' do
        it 'renders the locket cert' do
          cert = template.render(merged_manifest_properties)
          expect(cert.strip).to eq('the locket cert')
        end
      end

      describe 'when the property doesn\'t exist' do
        before do
          merged_manifest_properties['locket'].delete('client_cert')
        end

        it 'raises an error when there is no locket.client_cert defined' do
          expect {
            template.render(merged_manifest_properties)
          }.to raise_error Bosh::Template::UnknownProperty
        end
      end
    end

    describe 'locket.key' do
      let(:template) {job.template('config/certs/locket.key')}

      describe 'when the property exits' do
        it 'renders the locket cert' do
          cert = template.render(merged_manifest_properties)
          expect(cert.strip).to eq('the locket key')
        end
      end

      describe 'when the property doesn\'t exist' do
        before do
          merged_manifest_properties['locket'].delete('client_key')
        end

        it 'raises an error when there is no locket_client_key defined' do
          expect {
            template.render(merged_manifest_properties)
          }.to raise_error Bosh::Template::UnknownProperty
        end
      end
    end
  end
end
