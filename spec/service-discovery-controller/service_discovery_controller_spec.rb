require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'service-discovery-controller job template rendering' do
    let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
    let(:release) { ReleaseDir.new(release_path) }
    let(:job) { release.job('service-discovery-controller') }

    describe 'bpm.yml' do
      let(:template) { job.template('config/bpm.yml') }
      let(:config) { YAML.safe_load(template.render({}, consumes: [])) }

      it 'sets the open file descriptor limit' do
        expect(config['processes'][0].dig('limits', 'open_files')).to eq(65535)
      end
    end

    describe 'config.json' do
      let(:template) { job.template('config/config.json') }
      let(:config) { YAML.safe_load(template.render({}, consumes: [])) }
      let(:links) do
        [
          Link.new(
            name: 'nats',
            properties: {
              'nats' => {
                'user' => 'nats',
                'password' => '1234',
                'port' => '4222',
                'machines' => ['192.168.50.123'],
                'tls_enabled' => true,
                'ca_certs' => 'cert',
                'cert_chain' => 'cert_chain',
                'private_key' => 'private key'
              }
            }
          )
        ]
      end

      context 'when ips have leading 0s' do
        it 'address fails with a nice message' do
        merged_manifest_properties = {'address' => '127.0.0.01'}
          expect {
            template.render(merged_manifest_properties, consumes: links)
          }.to raise_error (/Invalid address/)
        end

        it 'log_level_address fails with a nice message' do
        merged_manifest_properties = {'log_level_address' => '127.0.0.01'}
          expect {
            template.render(merged_manifest_properties, consumes: links)
          }.to raise_error (/Invalid log_level_address/)
        end
      end
    end
  end
end
