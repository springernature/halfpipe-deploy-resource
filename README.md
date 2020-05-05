# cf-resource
This is a Concourse resource for doing zero downtime deploys to Cloudfoundry. 
Previously this worked in conjunction with the [halfpipe-cf-plugin](https://github.com/springernature/halfpipe-cf-plugin).
Now this project contains all behaviour. This resource also allows you to deploy docker-images to cf and to use the experimental rolling deploy feature of cf api version 7.


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

* `command`: _required_. The halfpipe-cf-plugin command to use. Must be one of `halfpipe-push`, `halfpipe-check`, `halfpipe-promote` or `halfpipe-cleanup`.
* `manifestPath`: _required_. Relative or absolute path to cf manifest.
* `appPath`: _required for halfpipe-push_. Relative or absolute path to the app bits you wish to deploy.
* `testDomain`: _required for halfpipe-push and halfpipe-promte_. Domain that will be used when constructing the candidate route for the app.
* `vars`: _optional_. Hash map containing environment variables that should be set on the application.
* `gitRefPath`: _optional_. Path to the `.git/ref` file. If this is set the app will get the environment variable `GIT_REVISION` set.
* `timeout`: _optional_. Timeout for each of the commands that the halfpipe cf plugin will execute.
* `preStartCommand`: _optional_. A CF command to run immediately before `cf start` in the `halfpipe-push` command. e.g. `cf events <app-name>`.
* `dockerUsername`: _optional_. The username to use when pushing a docker image to cf.
* `dockerPassword`: _optional_. The password to use when pushing a docker image to cf.
* `dockerTag`: _optional_. The dockertag to set or override the dockertag set in the cf manifest.
* `buildVersionPath`: _optional_. path to the versionfile. If this is set the app will get the environment variable `BUILD_VERSION` set.
* `instances`: _optional_. The number of instances to deploy when using the rolling deploy strategy.

 
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
        gitRefPath: my-apps-git-repo/.git/ref
        vars:
          EXTRA_VAR: "Yo, im a env var in the CF app"
          SECRET_VAR: ((some.secret))
    - put: cf-resource
      params:
        command: halfpipe-promote
        manifestPath: my-apps-git-repo/manifest.yml
        testDomain: some.random.domain.com
        timeout: 10m
    - put: cf-resource
      params:
        command: halfpipe-delete
        manifestPath: my-apps-git-repo/manifest.yml
```

# What do the different commands do?

## halfpipe-push

This simply deploys the application as `app-name-CANDIDATE` to a test route `app-name-{SPACE}-CANDIDATE.{DOMAIN}`

## halfpipe-check

Checks that all instances of the app is up and running, useful to stick between `halfpipe-push` and `halfpipe-promote`

## halfpipe-promote

* This binds all the routes from the manifest to the `app-name-CANDIDATE`
* Removes the test route from `app-name-CANDIDATE`
* renames `app-name-OLD` to `app-name-DELETE`
* renames `app-name` to `app-name-OLD` 
* renames `app-name-CANDIDATE` to `app-name`
* stops `app-name-OLD`

## halfpipe-cleanup

Simply deletes the app `app-name-DELETE`


