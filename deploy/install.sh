#!/usr/bin/env bash
set -euo pipefail

timestamp() {
  date +"%Y-%m-%d %T"
}

info() {
  local flag
  flag="$(timestamp)"
  echo -e "\033[36m INFO [$flag] >> $* \033[0m"
}

warn() {
  local flag
  flag="$(timestamp)"
  echo -e "\033[33m WARN [$flag] >> $* \033[0m"
}

error() {
  local flag
  flag="$(timestamp)"
  echo -e "\033[31m ERROR [$flag] >> $* \033[0m"
  exit 1
}

RELEASE_NAME="${RELEASE_NAME:-dataflow}"
NAMESPACE="${NAMESPACE:-dataflow-system}"
HELM_OPTS="${HELM_OPTS:-}"
DEFAULT_VALUES_FILE="./charts/dataflow/dataflow-values.yaml"
USER_VALUES_DIR="/root/.sealos/cloud/values/apps/dataflow"
USER_VALUES_FILE="${USER_VALUES_DIR}/dataflow-values.yaml"

get_sealos_config() {
  local key=$1
  kubectl get configmap sealos-config -n sealos-system -o "jsonpath={.data.${key}}"
}

decode_base64() {
  local raw=$1
  local decoded=""

  if decoded="$(printf '%s' "${raw}" | base64 --decode 2>/dev/null)"; then
    printf '%s' "${decoded}"
    return 0
  fi

  if decoded="$(printf '%s' "${raw}" | base64 -d 2>/dev/null)"; then
    printf '%s' "${decoded}"
    return 0
  fi

  return 1
}

get_secret_data() {
  local secret_name=$1
  local key=$2
  local encoded=""

  encoded="$(kubectl get secret "${secret_name}" -n "${NAMESPACE}" -o "jsonpath={.data.${key}}" 2>/dev/null || true)"
  [ -n "${encoded}" ] || return 1

  decode_base64 "${encoded}"
}

find_existing_dataflow_secret() {
  local name=""

  for name in "${RELEASE_NAME}-secret" "${RELEASE_NAME}-dataflow-secret" "dataflow-runtime"; do
    if kubectl get secret "${name}" -n "${NAMESPACE}" >/dev/null 2>&1; then
      echo "${name}"
      return 0
    fi
  done

  while IFS= read -r name; do
    [ -n "${name}" ] || continue
    if kubectl get secret "${name}" -n "${NAMESPACE}" >/dev/null 2>&1; then
      echo "${name}"
      return 0
    fi
  done < <(kubectl get secret -n "${NAMESPACE}" \
    -l "app.kubernetes.io/instance=${RELEASE_NAME},app.kubernetes.io/name=dataflow" \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)

  return 1
}

ensure_user_values_file() {
  mkdir -p "${USER_VALUES_DIR}"

  if [ ! -f "${USER_VALUES_FILE}" ]; then
    cp "${DEFAULT_VALUES_FILE}" "${USER_VALUES_FILE}"
    info "Generated default user values at ${USER_VALUES_FILE}"
    return 0
  fi

  info "Using user values from ${USER_VALUES_FILE}"
}

sealos_cloud_domain="$(get_sealos_config cloudDomain || true)"
[ -n "${sealos_cloud_domain}" ] || error "Failed to read sealos-config.data.cloudDomain"

session_key=""
session_key_source="generated"
existing_secret_name="$(find_existing_dataflow_secret || true)"

if [ -n "${existing_secret_name}" ]; then
  info "Found existing secret ${existing_secret_name}, trying to reuse WHODB_SESSION_ENCRYPTION_KEY"
  session_key="$(get_secret_data "${existing_secret_name}" "WHODB_SESSION_ENCRYPTION_KEY" || true)"

  if [ -n "${session_key}" ]; then
    session_key_source="secret:${existing_secret_name}"
  fi
fi

if [ -z "${session_key}" ]; then
  warn "WHODB_SESSION_ENCRYPTION_KEY not found in existing secret, generating a new one"
  session_key="$(openssl rand -hex 16)"
  session_key_source="generated"
fi

info "Session key source: ${session_key_source}"
ensure_user_values_file


helm_set_args=(
  --set-string "cloudDomain=${sealos_cloud_domain}"
  --set-string "session.encryptionKey=${session_key}"
)

helm_opts_arr=()
if [ -n "${HELM_OPTS}" ]; then
  # shellcheck disable=SC2206
  helm_opts_arr=(${HELM_OPTS})
fi

GLOBAL_VALUES_FILE="/root/.sealos/cloud/values/global.yaml"
if [ -f "$GLOBAL_VALUES_FILE" ]; then
  helm_set_args+=(
    -f "$GLOBAL_VALUES_FILE"
  )
fi

info "Installing chart charts/dataflow into namespace ${NAMESPACE}"
helm upgrade -i "${RELEASE_NAME}" -n "${NAMESPACE}" --create-namespace charts/dataflow \
  -f "${DEFAULT_VALUES_FILE}" \
  -f "${USER_VALUES_FILE}" \
  "${helm_set_args[@]}" \
  "${helm_opts_arr[@]}"
