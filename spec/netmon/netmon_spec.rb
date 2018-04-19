require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'netmon.json.erb' do
    describe 'template rendering' do
      let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
      let(:release) {ReleaseDir.new(release_path)}
      let(:template) {job.template('config/netmon.json')}

      describe 'netmon job' do
        let(:job) {release.job('netmon')}

        describe 'netmon.json' do
          describe 'when overriding defaults' do
            let(:merged_manifest_properties) do
              {
                'metron_address' => '123.123.123.1:2222',
                'poll_interval' => 40,
                'interface_name' => 'so-lame',
                'log_level' => 'fatal',
                'disable' => false,
              }
            end

            it 'creates a config/netmon.json from properties' do
              config = JSON.parse(template.render(merged_manifest_properties))
              expect(config).to eq({
                'metron_address' => '123.123.123.1:2222',
                'poll_interval' => 40,
                'interface_name' => 'so-lame',
                'log_level' => 'fatal',
                'log_prefix' => 'cfnetworking',
              })
            end
          end

          describe 'when accepting defaults' do
            let(:merged_manifest_properties) { {} }
            it 'creates a config/netmon.json from properties' do
              config = JSON.parse(template.render(merged_manifest_properties))
              expect(config).to eq({
                'metron_address' => '127.0.0.1:3457',
                'poll_interval' => 30,
                'interface_name' => 'silk-vtep',
                'log_level' => 'info',
                'log_prefix' => 'cfnetworking',
              })
            end
          end

          describe 'when disabled' do
            let(:merged_manifest_properties) { { 'disable' => true } }
            it 'creates a config/netmon.json from properties' do
              expect(template.render(merged_manifest_properties).strip).to be_empty
            end
          end
        end
      end
    end
  end
end
