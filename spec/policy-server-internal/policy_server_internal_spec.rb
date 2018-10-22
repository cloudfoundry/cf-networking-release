require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'policy-server-internal job template rendering' do
    let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
    let(:release) {ReleaseDir.new(release_path)}
    let(:job) {release.job('policy-server-internal')}

    let(:merged_manifest_properties) do
      {
        'disable' => false,
        'listen_ip' => '111.11.11.1',
        'debug_port' => 1234,
        'health_check_port' => 2345,
        'internal_listen_port' => 3456,
        'ca_cert' => 'meow-a-real-cert',
        'server_key' => 'password-please',
        'metron_port' => 4567,
        'log_level' => 'error',
        'database' => {
          'connect_timeout_seconds' => 30,
        },
        'max_idle_connections' => 4,
        'max_open_connections' => 5,
        'connections_max_lifetime_seconds' => 54,
        'enforce_experimental_dynamic_egress_policies' => true,
      }
    end

    let(:dbconn_host) {'some-database-host'}

    let(:db_properties) do
      {
        'type' => 'some-database-type',
        'username' => 'some-database-username',
        'password' => 'some-database-password',
        'port' => 4321,
        'name' => 'some-database-name',
        'host' => dbconn_host,
        'require_ssl' => true,
        'ca_cert' => 'some ca cert',
        'skip_hostname_validation' => true,
        'client_cert' => 'hello i am a cert',
        'client_key' => 'knock knock its your key',
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

    let(:tag_link) do
      Link.new(name: 'tag_length', instances: [LinkInstance.new()], properties: {'tag_length' => 1})
    end

    let(:db_link) do
      Link.new(name: 'database', instances: [LinkInstance.new(address: 'some-other-database-address')])
    end

    let(:links) {[dbconn_link, tag_link, db_link]}

    describe 'database_ca.crt' do
      let(:template) {job.template('config/certs/database_ca.crt')}
      it 'renders the cert' do
        cert = template.render(merged_manifest_properties, consumes: links)
        expect(cert).to eq('some ca cert')
      end
    end

    describe 'db_client.crt' do
      let(:template) {job.template('config/certs/db_client.crt')}
      it 'renders the cert' do
        cert = template.render(merged_manifest_properties, consumes: links)
        expect(cert).to eq("hello i am a cert\n")
      end
    end

    describe 'db_client.key' do
      let(:template) {job.template('config/certs/db_client.key')}
      it 'renders the key' do
        cert = template.render(merged_manifest_properties, consumes: links)
        expect(cert).to eq("knock knock its your key\n")
      end
    end

    describe 'policy-server-internal.json' do
      let(:template) {job.template('config/policy-server-internal.json')}

      it 'creates a config/policy-server-internal.json from properties' do
        config = JSON.parse(template.render(merged_manifest_properties, consumes: links))
        expect(config).to eq({
          'listen_host' => '111.11.11.1',
          'debug_server_port' => 1234,
          'health_check_port' => 2345,
          'internal_listen_port' => 3456,
          'database' => {
            'type' => 'some-database-type',
            'user' => 'some-database-username',
            'password' => 'some-database-password',
            'port' => 4321,
            'database_name' => 'some-database-name',
            'host' => 'some-database-host',
            'timeout' => 30,
            'require_ssl' => true,
            'ca_cert' => '/var/vcap/jobs/policy-server-internal/config/certs/database_ca.crt',
            'skip_hostname_validation' => true,
            'client_cert' => '/var/vcap/jobs/policy-server-internal/config/certs/db_client.crt',
            'client_key' => '/var/vcap/jobs/policy-server-internal/config/certs/db_client.key',
          },
          'max_idle_connections' => 4,
          'max_open_connections' => 5,
          'connections_max_lifetime_seconds' => 54,
          'tag_length' => 1,
          'metron_address' => '127.0.0.1:4567',
          'log_level' => 'error',
          'enforce_experimental_dynamic_egress_policies' => true,

          # hard-coded values, not exposed as bosh spec properties
          'debug_server_host' => '127.0.0.1',
          'log_prefix' => 'cfnetworking',
          'ca_cert_file' => '/var/vcap/jobs/policy-server-internal/config/certs/ca.crt',
          'server_cert_file' => '/var/vcap/jobs/policy-server-internal/config/certs/server.crt',
          'server_key_file' => '/var/vcap/jobs/policy-server-internal/config/certs/server.key',
          'request_timeout' => 5
          })
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
    end
  end
end
