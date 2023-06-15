# nginx-ect - nginx edge config test

This is a simple tool to check the impact of a change to an nginx configuration.

`nginx -t` will check the syntax of the configuration, but it won't tell you if the change will break your site.

## Usage

- Run `nginx-ect` before applying changes to your nginx configuration. `nginx-ect` will capture the state of all of the sites in your configuration and their current edge status.
- Apply your changes to your nginx configuration.
- Run `nginx-ect` again to see the impact of your changes.

```bash
Usage of nginx-ect:
  -c int
        concurrency (default 10)
  -d string
        input file to diff against state file
  -h    verify hash (default true)
  -i string
        input file
  -l string
        log level (default "debug")
  -s string
        state file (default "nginx-ect.state.json")
  -t string
        timeout (default "5s")
  -v    version
  -x string
        comma separated list of server names to exclude
```

## Example

```bash
$ nginx-ect -i /tmp/nginx.conf
# apply changes to nginx.conf
$ nginx-ect -d /tmp/nginx.conf
```

By default, the config file hash will be verified on diff, to ensure there haven't been any config changes between index and diff check. If you don't want to verify the hash, use the `-h=false` flag.