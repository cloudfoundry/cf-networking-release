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
        ]
      }
    end

    describe 'handlers.json' do
      let(:template) {job.template('dns/handlers.json')}

      it 'creates a dns/handlers.json from properties' do
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
          }
        ])
      end

      context 'when internal_domains is not set' do
        let(:merged_manifest_properties) { {} }

        it 'should render a json file with default apps.internal domain' do
          config = JSON.parse(template.render(merged_manifest_properties))
          expect(config).to eq([
            {
              'domain' => 'apps.internal.',
              'cache' => {'enabled' => false},
              'source' => {
                'type' => 'http',
                'url' => 'http://127.0.0.1:8053'
              }
            },
          ])
        end
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
