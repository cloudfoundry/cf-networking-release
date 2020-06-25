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
  end
end
