require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'proxy_plugin.json.erb' do
    describe 'template rendering' do
      let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
      let(:release) {ReleaseDir.new(release_path)}
      let(:merged_manifest_properties) { {} }
      let(:job) {release.job('proxy-plugin')}
      let(:template) {job.template('config/proxy-plugin.json')}
      let(:links) { [] }

      describe 'proxy-plugin job' do
        context 'when the cf-networking link is present' do
            let(:links){[
                Link.new(
                    name: 'cf_network',
                    instances: [LinkInstance.new()],
                    properties: { 'cf_networking' => { 'network' => '10.255.0.0/16' } }
                )
            ]}

          context 'when proxy_range is set' do
            let(:merged_manifest_properties) { { 'proxy_range' =>  '10.266.0.0/16' } }

            it 'chooses the proxy_range value' do
              clientConfig = JSON.parse(template.render(merged_manifest_properties, consumes: links))
              expect(clientConfig['proxy_range']).to eq('10.266.0.0/16')
            end

          end

          it 'uses the link value' do
            clientConfig = JSON.parse(template.render(merged_manifest_properties, consumes: links))
            expect(clientConfig['proxy_range']).to eq('10.255.0.0/16')
          end

          context 'when cf_networking.network nor proxy_range is set' do
            let(:links){[
                Link.new(
                    name: 'cf_network',
                    instances: [LinkInstance.new()],
                    properties: { 'cf_networking' => { 'meow' => 'pew pew' } }
                )
            ]}

            it 'errors' do
              expect {
                template.render(merged_manifest_properties, consumes: links)
              }.to raise_error("Must specify `proxy_range` property, or have it provided from the property `cf_networking.network` via bosh links")
            end
          end
        end

        context 'when the proxy_range is set' do
          let(:merged_manifest_properties) { { 'proxy_range' =>  '10.266.0.0/16' } }

          it 'uses that value' do
            clientConfig = JSON.parse(template.render(merged_manifest_properties, consumes: links))
            expect(clientConfig['proxy_range']).to eq('10.266.0.0/16')
          end
        end

        context 'when neither the link nor proxy_range is set' do
          it 'errors' do
            expect {
              template.render(merged_manifest_properties, consumes: links)
            }.to raise_error("Must specify `proxy_range` property, or have it provided from the property `cf_networking.network` via bosh links")
          end
        end
      end
    end
  end
end
