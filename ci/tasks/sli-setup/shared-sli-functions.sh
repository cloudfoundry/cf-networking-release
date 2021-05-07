#!/usr/bin/env bash

function get_var_from_json() {
  local name="${1}"
  jq -r ".${name}" "${ROOT}/variables/variables.json"
}

function cf_login() {
  local cf_username="$(get_var_from_json admin_user)"
  local cf_password="$(get_var_from_json admin_password)"
  local api="$(get_var_from_json api)"

    cf api --skip-ssl-validation "${api}"
    cf auth "${cf_username}" "${cf_password}"
}

function cf_target() {
    cf target -o "${SLI_ORG}" -s "${SLI_SPACE}"
}

function now_in_ms() {
    echo $(($(date +%s%N)/1000000))
}

function map_internal_route() {
    cf map-route "${SLI_APP_NAME}" apps.internal --hostname "${SLI_APP_NAME}"
}

function unmap_internal_route() {
    cf unmap-route "${SLI_APP_NAME}" apps.internal --hostname "${SLI_APP_NAME}"
}
