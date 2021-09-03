# Smoke Tests

The `cf-networking-release` smoke tests can be run in a production
environment to verify that basic features of cf-networking are
propertly functions.

In order to run the cf-networking smoke tests, you'll need to add
a smoke test user.

The user should have at minimum `network.write` privileges and
be added as an `OrgManager` for a persistent test org.

### Create a Smoke Test User

#### Option 1: Add smoke test user to your manifest

For example add the following to an opsfile:

```yaml
# smoke test user
- type: replace
  path: /instance_groups/name=uaa/jobs/name=uaa/properties/uaa/scim/users/-
    value:
			name: cf-networking-smokes
			password: "((cf_networking_smoke_user_password))"
			groups:
			- cloud_controller.read
			- cloud_controller.write
			- openid
			- scim.me
			- network.write

- type: replace
  path: /variables/-
  value:
    name: cf_networking_smoke_user_password
    type: password
```

#### Option 2: Use uaac to add the user

1. Target your environment

	```bash
	uaac target uaa.my-environment.com --skip-ssl-validation
	```

2. Authenticate with a user who can create users

	```bash
	uaac token client get
	 Client ID: admin
	 Client secret:  <uaa_admin_client_secret>
	```

3. Create a smoke test user (any name is fine)

	```bash
	$ uaac user add some-user-name --emails some-user-name
	Password: some-password
	Verify password: some-password
	user account successfully added
	```

4. Grant network.write for the user

	You may need to add the `network.write` group first:

	```bash
	uaac group add network.write
	```

	Then grant it for the user:

	```bash
	uaac member add network.write some-user-name
	```



### Create the Smoke Test org and add the user as OrgManager

```bash
cf auth admin <uaa_scim_users_admin_password>
cf create-org some-org
cf set-org-role smoke-test-user some-org OrgManager
```


### Configure and run smoke tests with user and org

```bash
pushd src/code.cloudfoundry.org/test/smoke/run_locally.sh
  export CONFIG=./smoke-config.json
  export APPS_DIR=../../example-apps
  echo '
  {
    "api": "api.my-environment.com",
    "smoke_user": "some-user-name",
    "smoke_password": "some-password",
    "app_instances": 4,
    "apps_domain": "my-environment.com",
    "prefix":"smoke-",
    "smoke_org": "some-org"
  }' > $CONFIG
  ginkgo -v .
popd
```
