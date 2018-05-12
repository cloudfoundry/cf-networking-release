require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'garden-cni job template rendering' do
    let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
    let(:release) {ReleaseDir.new(release_path)}
    let(:job) {release.job('garden-cni')}

    describe 'adapter.json.erb' do
      let(:template) {job.template('config/adapter.json')}

      describe 'when overriding defaults' do
        let(:merged_manifest_properties) do
          {
            'cni_plugin_dir' => 'meow-plugin-dir',
            'cni_config_dir' => 'meow-config-dir',
            'nat_port_range_start' => 1111,
            'nat_port_range_size' => 5555,
            'search_domains' => ['meow', 'woof', 'neopets'],
            'experimental_proxy_redirect_cidr' => 'some-proxy-cidr'
          }
        end

        it 'creates a config/adapter.json from properties' do
          clientConfig = JSON.parse(template.render(merged_manifest_properties))
          expect(clientConfig).to eq({
            'cni_plugin_dir' => 'meow-plugin-dir',
            'cni_config_dir' => 'meow-config-dir',
            'bind_mount_dir' => '/var/vcap/data/garden-cni/container-netns',
            'state_file' => '/var/vcap/data/garden-cni/external-networker-state.json',
            'start_port' => 1111,
            'total_ports' => 5555,
            'log_prefix' => 'cfnetworking',
            'search_domains' => ['meow', 'woof', 'neopets'],
            'iptables_lock_file' => '/var/vcap/data/garden-cni/iptables.lock',
            'proxy_redirect_cidr' => 'some-proxy-cidr',
            'proxy_port' => 16001,
            'proxy_uid' => 0,
          })
        end
      end

      describe 'when accepting defaults' do
        let(:merged_manifest_properties) { {} }

        it 'creates a config/adapter.json from defaults' do
          clientConfig = JSON.parse(template.render(merged_manifest_properties))
          expect(clientConfig).to eq({
            'cni_plugin_dir' => '/var/vcap/packages/cni/bin',
            'cni_config_dir' => '/var/vcap/jobs/cni/config/cni',
            'bind_mount_dir' => '/var/vcap/data/garden-cni/container-netns',
            'state_file' => '/var/vcap/data/garden-cni/external-networker-state.json',
            'start_port' => 61000,
            'total_ports' => 5000,
            'log_prefix' => 'cfnetworking',
            'search_domains' => [],
            'iptables_lock_file' => '/var/vcap/data/garden-cni/iptables.lock',
            'proxy_redirect_cidr' => '',
            'proxy_port' => 16001,
            'proxy_uid' => 0,
          })
        end
      end
    end
  end
end
