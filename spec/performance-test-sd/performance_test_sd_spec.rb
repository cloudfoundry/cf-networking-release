require 'rspec'
require 'bosh/template/test'
require 'json'

module Bosh::Template::Test
  describe 'performance-test-sd job template rendering' do
    let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
    let(:release) { ReleaseDir.new(release_path) }
    let(:job) { release.job('performance-test-sd') }
    let(:links) do
      [
        Link.new(
          name: 'nats-tls',
          instances: [LinkInstance.new(address: '10.0.0.5')],
          properties: {
            'nats' => {
              'user' => 'nats-user',
              'password' => 'nats-password',
              'port' => 4222,
              'monitor_port' => 8222,
              'external' => {
                'tls' => {
                  'ca' => 'nats-ca-pem'
                }
              }
            }
          }
        )
      ]
    end
    let(:properties) do
      {
        'nats' => {
          'cert_chain' => 'client-cert-pem',
          'private_key' => 'client-private-key-pem'
        }
      }
    end

    describe 'config.json' do
      let(:template) { job.template('config.json') }

      it 'renders nats-tls connection settings, certificate paths, and current load values' do
        config = JSON.parse(template.render(properties, consumes: links))

        expect(config).to eq({
          'nats_url' => '10.0.0.5',
          'nats_username' => 'nats-user',
          'nats_password' => 'nats-password',
          'nats_port' => 4222,
          'nats_monitoring_port' => 8222,
          'nats_ca_certs' => '/var/vcap/jobs/performance-test-sd/config/certs/nats_ca_certs.crt',
          'nats_cert_chain' => '/var/vcap/jobs/performance-test-sd/config/certs/nats_cert_chain.crt',
          'nats_private_key' => '/var/vcap/jobs/performance-test-sd/config/certs/nats_private_key.key',
          'num_messages' => 10,
          'num_publishers' => 10
        })
      end

      context 'when the nats-tls link provides nats.hostname' do
        before do
          links[0].properties['nats']['hostname'] = '10.0.255.190'
        end

        it 'dials the hostname so it matches the server certificate SAN' do
          config = JSON.parse(template.render(properties, consumes: links))

          expect(config['nats_url']).to eq('10.0.255.190')
        end
      end
    end

    describe 'certificate templates' do
      it 'renders the NATS CA from the link' do
        template = job.template('config/certs/nats_ca_certs.crt')
        expect(template.render(properties, consumes: links).strip).to eq('nats-ca-pem')
      end

      it 'renders the NATS client certificate from properties' do
        template = job.template('config/certs/nats_cert_chain.crt')
        expect(template.render(properties, consumes: links).strip).to eq('client-cert-pem')
      end

      it 'renders the NATS client private key from properties' do
        template = job.template('config/certs/nats_private_key.key')
        expect(template.render(properties, consumes: links).strip).to eq('client-private-key-pem')
      end
    end
  end
end
