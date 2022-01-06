require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'bosh-dns-adapter job template rendering' do
    let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
    let(:release) {ReleaseDir.new(release_path)}
    let(:job) {release.job('bosh-dns-adapter')}

    let(:merged_manifest_properties) do
      {
        'internal_domains' => [
          'my.internal.app.domain.',
          'other.internal.app.domain.'
        ],
        'internal_service_mesh_domains' => [
          'myistio.internal.app.domain.'
        ],
        'internal_route_vip_range' => '127.128.0.0/8',
      }
    end

    describe 'bpm.yml' do
      let(:config) {
        template = job.template('config/bpm.yml')
        YAML.safe_load(template.render({}, consumes: []))
      }

      it 'sets the open file descriptor limit' do
        expect(config['processes'][0]['limits']['open_files']).to eq(65_535)
      end
    end

    describe 'config.json' do
      let(:template) { job.template('config/config.json') }
      let(:links) do
        [
          Link.new(
            name: 'service-discovery-controller',
            properties: {
              'port' => 1234
            }
          ),
          Link.new(
            name: 'cloud_controller_container_networking_info',
            properties: {
              'cc' => {'internal_route_vip_range' => '192.168.0.1/24'}
            }
          )
        ]
      end

      it 'renders a file with default properties' do
        config = JSON.parse(template.render({}, consumes: links))
        expect(config).to eq({
          'address' => '127.0.0.1',
          'ca_cert' => '/var/vcap/jobs/bosh-dns-adapter/config/certs/server_ca.crt',
          'client_cert' => '/var/vcap/jobs/bosh-dns-adapter/config/certs/client.crt',
          'client_key' => '/var/vcap/jobs/bosh-dns-adapter/config/certs/client.key',
          'log_level_address' => '127.0.0.1',
          'log_level_port' => 8066,
          'metrics_emit_seconds' => 10,
          'metron_port' => 3457,
          'port' => '8053',
          'service_discovery_controller_address' => 'service-discovery-controller.service.cf.internal',
          'service_discovery_controller_port' => '1234',
          'internal_service_mesh_domains' => [],
          'internal_route_vip_range' => '192.168.0.1/24',
          'vip_resolver_address' => '',
        })
      end

      describe 'when the mesh domain has no trailing dot' do
        it 'appends a dot to the domain name' do
          properties = { 'internal_service_mesh_domains' => ['domain.with.no.trailing.dot'] }
          config = JSON.parse(template.render(properties, consumes: links))
          expect(config['internal_service_mesh_domains']).to eq(['domain.with.no.trailing.dot.'])
        end
      end

      describe 'with custom properties' do
        it 'renders a file with custom properties' do
          config = JSON.parse(template.render(merged_manifest_properties, consumes: links))
          expect(config).to eq({
            'address' => '127.0.0.1',
            'ca_cert' => '/var/vcap/jobs/bosh-dns-adapter/config/certs/server_ca.crt',
            'client_cert' => '/var/vcap/jobs/bosh-dns-adapter/config/certs/client.crt',
            'client_key' => '/var/vcap/jobs/bosh-dns-adapter/config/certs/client.key',
            'log_level_address' => '127.0.0.1',
            'log_level_port' => 8066,
            'metrics_emit_seconds' => 10,
            'metron_port' => 3457,
            'port' => '8053',
            'service_discovery_controller_address' => 'service-discovery-controller.service.cf.internal',
            'service_discovery_controller_port' => '1234',
            'internal_route_vip_range' => '127.128.0.0/8',
            'internal_service_mesh_domains' => ['myistio.internal.app.domain.'],
            'vip_resolver_address' => '',
          })
        end
      end

      describe 'when the optional vip_resolver_conn link is provided' do
        let(:links_with_vip_resolver_conn) do
          links << Link.new(
            name: 'vip_resolver_conn',
            properties: {
              'listen_port_for_vip_resolver' => 1234,
            },
            address: 'copilot.bosh',
          )
        end

        it 'renders the copilot address:port in the json' do
          template_str = template.render(merged_manifest_properties, consumes: links_with_vip_resolver_conn)
          config = JSON.parse(template_str)
          expect(config['vip_resolver_address']).to eq('copilot.bosh:1234')
        end
      end
    end

    describe 'handlers.json' do
      let(:template) {job.template('dns/handlers.json')}

      it 'creates a dns/handlers.json with default properties' do
        config = JSON.parse(template.render(merged_manifest_properties))
        expect(config).to eq([
          {
            'domain' => 'my.internal.app.domain.',
            'cache' => {'enabled' => false},
            'source' => {
              'type' => 'http',
              'url' => 'http://127.0.0.1:8053'
            }
          },
          {
            'domain' => 'other.internal.app.domain.',
            'cache' => {'enabled' => false},
            'source' => {
              'type' => 'http',
              'url' => 'http://127.0.0.1:8053'
            }
          },
          {
            'domain' => 'myistio.internal.app.domain.',
            'cache' => {'enabled' => false},
            'source' => {
              'type' => 'http',
              'url' => 'http://127.0.0.1:8053'
            }
          }
        ])
      end

      it 'creates a dns/handlers.json with custom properties' do
        properties = {
          'internal_domains' => ['hello.world'],
          'internal_service_mesh_domains' => ['helloistio.world'],
          'port' => 1001,
          'address' => '0.0.0.0'
        }
        config = JSON.parse(template.render(properties))
        expect(config).to eq([
          {
            'domain' => 'hello.world',
            'cache' => {'enabled' => false},
            'source' => {
              'type' => 'http',
              'url' => 'http://0.0.0.0:1001'
            }
          },
          {
            'domain' => 'helloistio.world',
            'cache' => {'enabled' => false},
            'source' => {
              'type' => 'http',
              'url' => 'http://0.0.0.0:1001'
            }
          }
        ])
      end

      context 'when cf_app_sd_disable is true' do
        let(:disabled_manifest_properties) do
        {
          'cf_app_sd_disable' => true,
        }
        end

        it 'should render an empty json file' do
          config = JSON.parse(template.render(disabled_manifest_properties))
          expect(config).to eq([])
        end
      end
    end
  end
end
