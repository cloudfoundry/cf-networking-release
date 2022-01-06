require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'policy-server job template rendering' do
    let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
    let(:release) {ReleaseDir.new(release_path)}
    let(:job) {release.job('policy-server')}
    let(:cc_hostname) {'some-cc-hostname'}
    let(:cc_port) {4567}
    let(:merged_manifest_properties) do
      {
        'disable' => false,
        'policy_cleanup_interval' => 1,
        'max_policies_per_app_source' => 2,
        'enable_space_developer_self_service' => true,
        'listen_ip' => '111.11.11.1',
        'listen_port' => 1234,
        'debug_port' => 2345,
        'enable_tls' => true,
        'server_cert' => 'the server cert',
        'server_key'=> 'the server key',
        'uaa_client' => 'some-uaa-client',
        'uaa_client_secret' => 'some-uaa-client-secret',
        'uaa_ca' => 'some-uaa-ca',
        'uaa_hostname' => 'some-uaa-hostname',
        'uaa_port' => 3456,
        'cc_hostname' => cc_hostname,
        'cc_port' => cc_port,
        'skip_ssl_validation' => true,
        'database' => {
          'type' => 'postgres',
          'host' => 'some-database-host',
          'username' => 'some-database-username',
          'password' => 'some-database-password',
          'port' => 5678,
          'name' => 'some-database-name',
          'require_ssl' => true,
          'ca_cert' => 'some-ca-cert',
          'connect_timeout_seconds' => 3,
          'skip_hostname_validation' => true,
        },
        'max_open_connections' => 5,
        'connections_max_lifetime_seconds' => 45,
        'tag_length' => 2,
        'metron_port' => 6789,
        'log_level' => 'debug',
        'allowed_cors_domains' => ['some-cors-domain'],
        'locket_ca_cert' => 'the locket ca cert',
        'locket_client_cert' => 'the locket cert',
        'locket_client_key' => 'the locket key',
      }
    end

    describe 'database_ca.crt' do
      let(:template) {job.template('config/certs/database_ca.crt')}
      it 'writes the content of database.ca_cert' do
        merged_manifest_properties['database']['ca_cert'] = 'the ca cert'
        expect(template.render(merged_manifest_properties).rstrip).to eq('the ca cert')
      end
    end

    describe 'server.crt' do
      let(:template) {job.template('config/certs/server.crt')}

      it 'renders the server cert' do
        cert = template.render(merged_manifest_properties)
        expect(cert.strip).to eq('the server cert')
      end

      describe 'when the property doesn\'t exist' do
        before do
          merged_manifest_properties.delete('server_cert')
        end
        describe 'when enable_tls is true' do
          before do
            merged_manifest_properties['enable_tls'] = true
          end

          it 'should err' do
            expect { template.render(merged_manifest_properties) }.to raise_error Bosh::Template::UnknownProperty
          end
        end
        describe 'when enable_tls is false' do
          before do
            merged_manifest_properties['enable_tls'] = false
          end

          it 'should not err' do
            expect { template.render(merged_manifest_properties) }.not_to raise_error
          end
        end
      end
    end

    describe 'server.key' do
      let(:template) {job.template('config/certs/server.key')}

      it 'renders the server key' do
        key = template.render(merged_manifest_properties)
        expect(key.strip).to eq('the server key')
      end

      describe 'when the property doesn\'t exist' do
        before do
          merged_manifest_properties.delete('server_key')
        end
        describe 'when enable_tls is true' do
          before do
            merged_manifest_properties['enable_tls'] = true
          end

          it 'should err' do
            expect { template.render(merged_manifest_properties) }.to raise_error Bosh::Template::UnknownProperty
          end
        end
        describe 'when enable_tls is false' do
          before do
            merged_manifest_properties['enable_tls'] = false
          end

          it 'should not err' do
            expect { template.render(merged_manifest_properties) }.not_to raise_error
          end
        end
      end
    end

    describe 'policy-server.json' do
      let(:template) {job.template('config/policy-server.json')}

      it 'creates a config/policy-server.json from properties' do
        config = JSON.parse(template.render(merged_manifest_properties))
        expect(config).to eq({
          'listen_host' => '111.11.11.1',
          'listen_port' => 1234,
          'log_prefix' => 'cfnetworking',
          'debug_server_host' => '127.0.0.1',
          'enable_tls' => true,
          'ca_cert_file' => '/var/vcap/jobs/policy-server/config/certs/server_ca.crt',
          'server_cert_file' => '/var/vcap/jobs/policy-server/config/certs/server.crt',
          'server_key_file'=> '/var/vcap/jobs/policy-server/config/certs/server.key',
          'debug_server_port' => 2345,
          'uaa_client' => 'some-uaa-client',
          'uaa_client_secret' => 'some-uaa-client-secret',
          'uaa_url' => 'https://some-uaa-hostname',
          'uaa_port' => 3456,
          'cc_ca_cert' => '/var/vcap/jobs/policy-server/config/certs/cc_ca.crt',
          'cc_url' => 'http://some-cc-hostname:4567',
          'skip_ssl_validation' => true,
          'database' => {
            'type' => 'postgres',
            'user' => 'some-database-username',
            'password' => 'some-database-password',
            'host' => 'some-database-host',
            'port' => 5678,
            'timeout' => 3,
            'database_name' => 'some-database-name',
            'require_ssl' => true,
            'ca_cert' => '/var/vcap/jobs/policy-server/config/certs/database_ca.crt',
            'skip_hostname_validation' => true,
          },
          'database_migration_timeout' => 600,
          'max_idle_connections' => 10,
          'max_open_connections' => 5,
          'connections_max_lifetime_seconds' => 45,
          'tag_length' => 2,
          'metron_address' => '127.0.0.1:6789',
          'log_level' => 'debug',
          'cleanup_interval' => 60,
          'max_policies' => 2,
          'enable_space_developer_self_service' => true,
          'allowed_cors_domains' => ['some-cors-domain'],
          'uaa_ca' => '/var/vcap/jobs/policy-server/config/certs/uaa_ca.crt',
          'request_timeout' => 5,
          'asg_sync_interval' => 60,
          'locket_address' => 'locket.service.cf.internal:8891',
          'locket_ca_cert_file' => '/var/vcap/jobs/policy-server/config/certs/locket_ca.crt',
          'locket_client_cert_file' => '/var/vcap/jobs/policy-server/config/certs/locket.crt',
          'locket_client_key_file' => '/var/vcap/jobs/policy-server/config/certs/locket.key',
        })
      end


      context 'when capi provides a link to the https endpoint' do
        let(:links) do
          [
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
          ]
        end

        before do
          merged_manifest_properties.delete('cc_hostname')
          merged_manifest_properties.delete('cc_port')
        end

        it 'uses the values from the cloud controller link' do
          policyServerJSON = JSON.parse(template.render(merged_manifest_properties, consumes: links))

          expect(policyServerJSON['cc_url']).to eq 'https://cc.service.internal:443'
          expect(policyServerJSON['cc_ca_cert']).to eq '/var/vcap/jobs/policy-server/config/certs/cc_ca.crt'
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
        let(:links) do
          [
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
          ]
        end

        it 'uses the property values, so the link can be overridden' do
          policyServerJSON = JSON.parse(template.render(merged_manifest_properties, consumes: links))

          expect(policyServerJSON['cc_url']).to eq 'http://use.me.pls:1234'
          expect(policyServerJSON['cc_ca_cert']).to eq '/var/vcap/jobs/policy-server/config/certs/cc_ca.crt'
        end
      end

      context 'when neither the cc_hostname nor cc_port property values are provided nor is the link provided' do
        before do
          merged_manifest_properties.delete('cc_hostname')
          merged_manifest_properties.delete('cc_port')
        end

        it 'should raise an error' do
          expect {
            JSON.parse(template.render(merged_manifest_properties))
          }.to raise_error '`cc_hostname` and `cc_port` properties were not supplied as manifest properties, nor were found in `cloud_controller_https_endpoint` link'
        end
      end

      context 'when tag length is valid' do
        [1, 2, 3].each do |i|
          it "does not raise when tag length is #{i}" do
            merged_manifest_properties['tag_length'] = i
            expect {
              JSON.parse(template.render(merged_manifest_properties))
            }.to_not raise_error
          end
        end
      end

      it 'raises an error when the tag length is too high' do
        merged_manifest_properties['tag_length'] = 4
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('tag length must be greater than 0 and less than 4')
      end

      it 'raises an error when the tag length is too low' do
        merged_manifest_properties['tag_length'] = 0
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('tag length must be greater than 0 and less than 4')
      end

      it 'raises an error when the driver (type) is unknown' do
        merged_manifest_properties['database']['type'] = 'bar'
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('unknown driver bar')
      end

      it 'raises an error when the driver (type) is missing' do
        merged_manifest_properties['database'].delete('type')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.type must be specified')
      end

      it 'raises an error when missing username' do
        merged_manifest_properties['database'].delete('username')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.username must be specified')
      end

      it 'raises an error when missing password' do
        merged_manifest_properties['database'].delete('password')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.password must be specified')
      end

      it 'raises an error when missing port' do
        merged_manifest_properties['database'].delete('port')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.port must be specified')
      end

      it 'raises an error when missing name' do
        merged_manifest_properties['database'].delete('name')
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('database.name must be specified')
      end

      it 'raises an error when the cleanup interval is too short' do
        merged_manifest_properties['policy_cleanup_interval'] = 0.7
        expect {
          JSON.parse(template.render(merged_manifest_properties))
        }.to raise_error('policy_cleanup_interval must be at least 1 minute')
      end

      it 'raises an error when asg_sync_enabled is true and the asg_sync_interval is invalid' do
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
        merged_manifest_properties['asg_sync_enabled'] = true
        intervals.each do |interval|
          merged_manifest_properties['asg_sync_interval'] = interval
          expect {
            JSON.parse(template.render(merged_manifest_properties))
          }.to raise_error('asg_sync_interval must be an integer greater than 0')
        end
      end
      it 'raises an error when asg_sync_enabled is true and there is no locket_address defined' do
        addrs = [
          '',
          'my-site-without-port.com',
          'http://asdf.com',
          'http://asdf:1234',
          'asdf.com:badport',
          'me+you:1234',
        ]
        merged_manifest_properties['asg_sync_enabled'] = true
        addrs.each do |addr|
          merged_manifest_properties['locket_address'] = addr
          expect {
            JSON.parse(template.render(merged_manifest_properties))
          }.to raise_error('asg_sync_enabled is true but the locket_address is invalid')
        end
      end
      it 'allows common domain name/ip addr combos for locket_address' do
        addrs = [
          'test.com:1234',
          '10.10.10.10:1234',
          'my-cool-site.com:1234',
        ]
        merged_manifest_properties['asg_sync_enabled'] = true
        addrs.each do |addr|
          merged_manifest_properties['locket_address'] = addr
          expect {
          JSON.parse(template.render(merged_manifest_properties))
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
          merged_manifest_properties.delete('locket_ca_cert')
        end

        it 'raises an error when asg_sync_enabled is true and there is no locket_ca_cert defined' do
          merged_manifest_properties['asg_sync_enabled'] = true
          expect {
            template.render(merged_manifest_properties)
          }.to raise_error Bosh::Template::UnknownProperty
        end

        it 'raises and error when asg_sync_enabled is false and there is no locket_ca_cert defined' do
          merged_manifest_properties['asg_sync_enabled'] = false
          expect {
            template.render(merged_manifest_properties)
          }.to_not raise_error
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
          merged_manifest_properties.delete('locket_client_cert')
        end

        it 'raises an error when asg_sync_enabled is true and there is no locket_client defined' do
          merged_manifest_properties['asg_sync_enabled'] = true
          expect {
            template.render(merged_manifest_properties)
          }.to raise_error Bosh::Template::UnknownProperty
        end

        it 'raises and error when asg_sync_enabled is false and there is no locket_client defined' do
          merged_manifest_properties['asg_sync_enabled'] = false
          expect {
            template.render(merged_manifest_properties)
          }.to_not raise_error
        end
      end
    end
    describe 'locket.key' do
      let(:template) {job.template('config/certs/locket.key')}
      describe 'When the property exits' do
        it 'renders the locket cert' do
          cert = template.render(merged_manifest_properties)
          expect(cert.strip).to eq('the locket key')
        end
      end

      describe 'when the property doesn\'t exist' do
        before do
          merged_manifest_properties.delete('locket_client_key')
        end

        it 'raises an error when asg_sync_enabled is true and there is no locket_client_key defined' do
          merged_manifest_properties['asg_sync_enabled'] = true
          expect {
            template.render(merged_manifest_properties)
          }.to raise_error Bosh::Template::UnknownProperty
        end

        it 'raises and error when asg_sync_enabled is false and there is no locket_client_key defined' do
          merged_manifest_properties['asg_sync_enabled'] = false
          expect {
            template.render(merged_manifest_properties)
          }.to_not raise_error
        end
      end
    end
  end
end
