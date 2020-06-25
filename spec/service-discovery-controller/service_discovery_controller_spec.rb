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

      context 'when open_files is not set' do
        let(:config) { YAML.safe_load(template.render({}, consumes: [])) }

        it 'does not set the open file descriptor limit' do
          expect(config['processes'][0].dig('limits', 'open_files')).to be_nil
        end
      end

      context 'when open_files is set' do
        let(:config) {
          YAML.safe_load(template.render(
            {
              'open_files' => 4096
            },
            consumes: []
          ))
        }

        it 'does not set the open file descriptor limit' do
          expect(config['processes'][0].dig('limits', 'open_files')).to eq(4096)
        end
      end
    end
  end
end
