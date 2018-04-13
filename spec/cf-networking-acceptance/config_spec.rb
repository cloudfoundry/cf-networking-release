require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'


module Bosh::Template::Test
  describe 'config.json.erb' do
    describe 'template rendering' do
      let(:release_path) {File.join(File.dirname(__FILE__), '../..')}
      let(:release) {ReleaseDir.new(release_path)}
      let(:merged_manifest_properties) do
        {
          'system_domain' => "some.system.domain",
          'apps_domain' => "some.apps.domain",
          'skip_ssl_validation' => true,
          'admin_user' => "some-admin-user",
          'admin_password' => "some-admin-password",
          'admin_client' => "some-admin-client",
          'admin_secret' => "some-admin-secret",
          'default_security_groups' => ["some-security-group"],
          'prefix' => "some-prefix-",
          'proxy_applications' => 1,
          'proxy_instances' => 2,
          'num_apps' => 3,
          'num_app_instances' => 4,
        }
      end

      describe 'cf-networking-acceptance/config.json job' do
        let(:job) {release.job('cf-networking-acceptance')}

        describe 'config.json' do
          let(:template) {job.template('config.json')}

          it 'creates a config.json from properties' do
            clientConfig = JSON.parse(template.render(merged_manifest_properties))
            expect(clientConfig).to eq({
              'api' => "api.some.system.domain",
              'apps_domain' => "some.apps.domain",
              'skip_ssl_validation' => true,
              'admin_user' => "some-admin-user",
              'admin_password' => "some-admin-password",
              'admin_client' => "some-admin-client",
              'admin_secret' => "some-admin-secret",
              'default_security_groups' => ["some-security-group"],
              'proxy_applications' => 1,
              'proxy_instances' => 2,
              'test_applications' => 3,
              'test_app_instances' => 4,
              'extra_listen_ports' => 2,
              'prefix' => "some-prefix-",
            })
          end
        end
      end
    end
  end
end
