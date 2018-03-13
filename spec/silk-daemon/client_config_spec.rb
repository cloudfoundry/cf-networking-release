require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'


module Bosh::Template::Test
  describe 'client-config.json.erb' do
    describe 'template rendering' do
      let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
      let(:release) {ReleaseDir.new(release_path)}
      let(:merged_manifest_properties) do
        {}
      end

      links = [
        Link.new(
          name: 'cf_network',
          instances: [LinkInstance.new()],
          properties: {
            'cf_networking' => {
              'network' => '10.255.0.0/16',
              'subnet_prefix_length' => 24
            }
          }
        )
      ]

      describe 'silk-daemon job' do let(:job) {release.job('silk-daemon')}
        describe 'config/client-config.json' do
          let(:template) {job.template('config/client-config.json')}

          context 'when temporary_vxlan_interface and vxlan_network are set' do
            let(:merged_manifest_properties) do
              {
                'cf_networking' => {
                  'silk_daemon' => {
                    'temporary_vxlan_interface' => 'some-vxlan-interface',
                    'vxlan_network' => 'some-vxlan-network'
                  }
                }
              }
            end

            it 'throws a helpful error' do
              expect {
                template.render(merged_manifest_properties, consumes: links)
              }.to raise_error("Cannot specify both 'temporary_vxlan_interface' and 'vxlan_network' properties.")
            end
          end

          context 'when temporary_vxlan_interface is set' do
            let(:merged_manifest_properties) do
              {
                'cf_networking' => {
                  'silk_daemon' => {
                    'temporary_vxlan_interface' => 'some-vxlan-interface',
                  }
                }
              }
            end

            it 'sets vxlan_interface_name' do
              clientConfig = JSON.parse(template.render(merged_manifest_properties, consumes: links))
              expect(clientConfig['vxlan_interface_name']).to eq("some-vxlan-interface")
            end
          end

          context 'when vxlan_network is set' do
            let(:merged_manifest_properties) do
              {
                'cf_networking' => {
                  'silk_daemon' => {
                    'vxlan_network' => 'fake-network'
                  }
                }
              }
            end
            networks = { 'fake-network' => { 'fake-network-settings' => {}, 'ip' => "192.74.65.4" } }
            spec = InstanceSpec.new(address: 'cloudfoundry.org', bootstrap: true, networks: networks)

            it 'sets the underlay_ip to the ip associated with vxlan_network' do
              clientConfig = JSON.parse(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(clientConfig['underlay_ip']).to eq("192.74.65.4")
            end
          end
        end
      end
    end
  end
end
