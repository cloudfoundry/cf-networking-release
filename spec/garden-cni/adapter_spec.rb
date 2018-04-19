require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'


module Bosh::Template::Test
  describe 'adapter.json.erb' do
    describe 'template rendering' do
      let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
      let(:release) {ReleaseDir.new(release_path)}
      let(:merged_manifest_properties) do
        {
          'cni_plugin_dir' => "meow-plugin-dir",
          'cni_config_dir' => "meow-config-dir",
          'nat_port_range_start' => 1111,
          'nat_port_range_size' => 5555,
          'search_domains' => ["meow", "woof", "neopets"]
        }
      end

      describe 'garden-cni job' do
        let(:job) {release.job('garden-cni')}

        describe 'adapter.json' do
          let(:template) {job.template('config/adapter.json')}

          it 'creates a config/adapter.json from properties' do
            clientConfig = JSON.parse(template.render(merged_manifest_properties))
            expect(clientConfig).to eq({
              "cni_plugin_dir" => "meow-plugin-dir",
              "cni_config_dir" => "meow-config-dir",
              "bind_mount_dir" => "/var/vcap/data/garden-cni/container-netns",
              "state_file" => "/var/vcap/data/garden-cni/external-networker-state.json",
              "start_port" => 1111,
              "total_ports" => 5555,
              "log_prefix" => "cfnetworking",
              "search_domains" => ["meow", "woof", "neopets"]
            })
          end
        end
      end
    end
  end
end
