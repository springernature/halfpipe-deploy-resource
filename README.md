# cf-resource
This is a Concourse resource for doing zero downtime deploys to CF using [halfpipe-cf-plugin](https://github.com/springernature/halfpipe-cf-plugin)

# Deploying to Concourse

You can use the docker image by defining the [resource type](https://concourse-ci.org/resource-types.html) in your pipeline YAML.

For example:

```
resource_types:
- name: cf-resource
  type: docker-image
  source:
    repository: platformengineering/cf-resource
    tag: stable
```

If you want to use a specific version of the halfpipe-push command simply set `version` to a tag version [from here](https://github.com/springernature/halfpipe-cf-plugin/releases)

# Source Configuration

* `api`: _required_. The CF API you wish to deploy to.
* `org`: _required_. The Org the app should be deployed in.
* `space`: _required_. The Space the app should be deployed into.
* `username`: _required_. The username for the user to use when deploying.
* `password`: _required_. The password for the user to use when deploying.
* `prometheusGatewayURL`: _optional_. If this is set metrics will be sent to Prometheus

### Example
```
resources:
- name: cf-resource
  type: cf-resource
  source:
    api: ((cloudfoundry.api-dev))
    org: my-org
    space: my-space
    username: ((cloudfoundry.username))
    password: ((cloudfoundry.password))
```

# Behavior

## `check`
Does nothing

## `in`
Does nothing

## `out`: deploys to CF.

Deploys app to cf

#### Parameters

* `command`: _required_. The halfpipe-cf-plugin command to use. Must be one of `halfpipe-push`, `halfpipe-promote` or `halfpipe-cleanup`.
* `manifestPath`: _required_. Relative or absolute path to cf manifest.
* `appPath`: _required for halfpipe-push_. Relative or absolute path to the app bits you wish to deploy.
* `testDomain`: _required for halfpipe-push and halfpipe-promte_. Domain that will be used when constructing the candidate route for the app.
* `space`: _required for halfpipe-push and halfpipe-promte_. Space will be used when constructing the candidate test route. 
* `vars`: _optional_. Hash map containing environment variables that should be set on the application.
* `gitRefPath`: _optional_. Path to the `.git/ref` file. If this is set the app will get the environment variable `GIT_REVISION` set
* `timeout`: _optional_. Timeout for each of the commands that the halfpipe cf plugin will execute
 
### Example
```
jobs:
- name: deploy-to-dev
  plan:
    - get: my-apps-git-repo
    - put: cf-resource
      params:
        appPath: my-apps-git-repo/target/distribution/artifact.zip
        command: halfpipe-push
        manifestPath: my-apps-git-repo/manifest.yml
        testDomain: some.random.domain.com
        space: dev
        gitRefPath: my-apps-git-repo/.git/ref
        vars:
          EXTRA_VAR: "Yo, im a env var in the CF app"
          SECRET_VAR: ((some.secret))
    - put: cf-resource
      params:
        command: halfpipe-promote
        manifestPath: my-apps-git-repo/manifest.yml
        testDomain: some.random.domain.com
        space: dev
        timeout: 10m
    - put: cf-resource
      params:
        command: halfpipe-delete
        manifestPath: my-apps-git-repo/manifest.yml
```