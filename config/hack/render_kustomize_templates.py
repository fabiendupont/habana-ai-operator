#!/usr/bin/env python3

import base64
import jinja2
import json
import os
import re
import requests
import urllib.parse

TEMPLATES = (
    {
        "tplfile": "config/hack/templates/manager/patches/env_images.yaml.j2",
        "target": "config/manager/patches/env_images.yaml"
    },
    {
        "tplfile": "config/hack/templates/manager/patches/controller_image.yaml.j2",
        "target": "config/manager/patches/controller_image.yaml"
    },
    {
        "tplfile": "config/hack/templates/manifests/patches/controller_image.yaml.j2",
        "target": "config/manifests/patches/controller_image.yaml"
    },
    {
        "tplfile": "config/hack/templates/manifests/patches/related_images.yaml.j2",
        "target": "config/manifests/patches/related_images.yaml"
    },
    {
        "tplfile": "config/hack/templates/manifests/patches/version.yaml.j2",
        "target": "config/manifests/patches/version.yaml"
    },
)

REDHAT_REGISTRIES = (
    "registry.redhat.io",
    "registry.access.redhat.com"
)

def _get_base_path():
    try:
        path = os.environ["GITHUB_WORKSPACE"]
    except:
        path = ""

    return path


def _get_operator_version():
    try:
        version = os.environ["VERSION"]
    except:
        version = "99.0.0"

    return version


def _get_digest_from_redirect_url(url):
    digest = os.path.basename(urllib.parse.urlparse(url).path)
    if not digest.startswith("sha256:"):
        print("The digest is not in the Red Hat registry redirect URL.")
        exit(1)

    return digest


def _get_image_digest(registry, namespace, name, tag):
    manifest_url = "https://%s/v2/%s/%s/manifests/%s" % (
            registry, namespace, name, tag)

    # We request the manifest v2 format in JSON
    manifest_headers = {
        "Accept": "application/vnd.docker.distribution.manifest.v2+json"
    }

    # With Red Hat registries, the redirect URL contains the digest
    allow_redirects = registry not in REDHAT_REGISTRIES

    manifest_resp = requests.get(manifest_url, headers=manifest_headers, allow_redirects=allow_redirects)
    if manifest_resp.status_code != 200:
        # Put all headers keys to lower case
        manifest_resp_headers = dict((k.lower(), v) for k, v in manifest_resp.headers.items())

        # Unauthenticated Red Hat registry
        if registry in REDHAT_REGISTRIES and manifest_resp.status_code == 302:
            return _get_digest_from_redirect_url(manifest_resp_headers['location'])

        # Fail if error is not related to authentication
        if manifest_resp.status_code != 401:
                print('Unexpected error fetching manifest: [%s] %s' % (manifest_resp.status_code, manifest_resp.reason))
                exit(1)

        # The status code is necessarily 401. So, let's get a token.
        if 'www-authenticate' not in manifest_resp_headers:
            print("Registry requires authentication, but doesn't tell which mechanism.")
            exit(1)

        if not manifest_resp_headers['www-authenticate'].lower().startswith("bearer "):
            print("Registry doesn't support bearer token authentication.")
            exit(1)

        auth = {}
        for i in list(filter(None, re.split(r'([^=]+="[^=]+"),?', re.sub('^[Bb]earer ', '', manifest_resp_headers['www-authenticate'])))):
            kv = i.split("=")
            auth[kv[0]] = re.sub('"', '', kv[1])

        token_url = "%s?scope=%s&service=%s" % (
                auth['realm'], auth['scope'], registry)
        print("Registry requires a token: %s" % token_url)

        registry_env = re.sub('\.', '_', registry).upper()
        basic_auth = {}
        for v in ["username", "password"]:
            try:
                basic_auth[v] = os.environ["%s_%s" % (registry_env, v.upper())]
            except:
                print('The environment variable "%s_%s" is not set.' % (registry_env, v.upper()))
                exit(1)

        # Encode credentials and put them in Authorization header
        token_headers = {}
        if basic_auth["username"] != "" or basic_auth["password"] != "":
            creds_str = "%s:%s" % (basic_auth["username"], basic_auth["password"])
            creds_bytes = creds_str.encode('ascii')
            creds_b64_bytes = base64.b64encode(creds_bytes)
            creds_header = creds_b64_bytes.decode('ascii')
            token_headers["Authorization"] = "Basic %s" % creds_header
            
        token_resp = requests.get(token_url, headers=token_headers)
        if token_resp.status_code != 200:
            print("Unexpected HTTP error: [%s] %s" % (token_resp.status_code, token_resp.reason))

        token = json.loads(token_resp.content)['token']

        manifest_headers["Authorization"] = "Bearer %s" % token

        manifest_resp = requests.get(manifest_url, headers=manifest_headers, allow_redirects=allow_redirects)
        if manifest_resp.status_code != 200:
            # Authenticated Red Hat registry
            if registry in REDHAT_REGISTRIES or manifest_resp.status_code == 302:
                return _get_digest_from_redirect_url(manifest_resp.headers['Location'])

            print("Unexpect HTTP error: [%s] %s" % (manifest_resp.status_code, manifest_resp.reason))
            exit(1)

    # Return the digest from the response headers is available
    if 'docker-content-digest' in manifest_resp.headers:
        return manifest_resp.headers['docker-content-digest']

    # Last resort is to take the digest from the response body
    manifest = json.loads(manifest_resp.content)
    if 'config' in manifest and 'digest' in manifest['config']:
        return manifest['config']['digest']

    return None


def _get_render_vars():
    # Retrieve the images specs from JSON config file
    try:
        images_file = os.environ['CONFIG_CONTAINER_IMAGES']
    except:
        print('The environment variable "CONFIG_CONTAINER_IMAGES" is not set.')
        exit()

    with open(images_file, 'r') as f:
        images = json.load(f)

    render_vars = {}
    for image in images:
        digest = _get_image_digest(image['registry'], image['namespace'], image['name'], image['tag'])
        if digest is None:
            render_vars[image['tplvar']] = "%s/%s/%s:%s" % (
                image['registry'], image['namespace'], image['name'], image['tag'])
        else:
            render_vars[image['tplvar']] = "%s/%s/%s@%s" % (
                image['registry'], image['namespace'], image['name'], digest)

        print("%s: %s" % (image['tplvar'], render_vars[image['tplvar']]))

    try:
        render_vars['CONTROLLER_MANAGER_IMAGE'] = os.environ['IMG']
    except:
        print('The environment variable "IMG" is not set')
        exit()

    render_vars['VERSION'] = _get_operator_version()

    return render_vars


def render_templates():
    base_path = _get_base_path()
    render_vars = _get_render_vars()

    for tpl in TEMPLATES:
        template_file_abs = os.path.abspath(os.path.join(base_path, tpl["tplfile"]))
        template_file_name = os.path.basename(template_file_abs)
        template_file_path = os.path.dirname(template_file_abs)

        rendered_file_abs = os.path.join(base_path, tpl["target"])
        rendered_file_name = os.path.basename(rendered_file_abs)
        rendered_file_path = os.path.dirname(rendered_file_abs)

        template_loader = jinja2.FileSystemLoader(template_file_path)
        environment = jinja2.Environment(loader=jinja2.FileSystemLoader(template_file_path))
        output_text = environment.get_template(template_file_name).render(render_vars)

        with open(rendered_file_abs, "w") as output_file:
            output_file.write(output_text)


if __name__ == "__main__":
    render_templates()
