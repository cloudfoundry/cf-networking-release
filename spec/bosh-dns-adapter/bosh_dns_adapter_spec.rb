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
        ]
      }
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
