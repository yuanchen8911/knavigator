#!/usr/bin/env bash
# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

failed=()

if [[ "${VERIFY_ENDS_NEWLINE:-true}" == "true" ]]; then
  echo "[*] Verifying ends newline..."
  "${ROOT_DIR}"/hack/verify-ends-newline.sh || failed+=(ends-newline)
fi

if [[ "${VERIFY_GO_MOD:-true}" == "true" ]]; then
  echo "[*] Verifying go mod..."
  "${ROOT_DIR}"/hack/verify-go-mod.sh || failed+=(go-mod)
fi

if [[ "${VERIFY_GO_FORMAT:-true}" == "true" ]]; then
  echo "[*] Verifying go format..."
  "${ROOT_DIR}"/hack/verify-go-format.sh || failed+=(go-format)
fi

if [[ "${VERIFY_GO_LINT:-true}" == "true" ]]; then
  echo "[*] Verifying go lint..."
  "${ROOT_DIR}"/hack/verify-go-lint.sh || failed+=(go-lint)
fi

if [[ "${VERIFY_YAMLLINT:-true}" == "true" ]]; then
  echo "[*] Verifying YAML lint..."
  "${ROOT_DIR}"/hack/verify-yamllint.sh || failed+=(yamllint)
fi

if [[ "${VERIFY_SHELLCHECK:-true}" == "true" ]]; then
  echo "[*] Verifying shell check..."
  "${ROOT_DIR}"/hack/verify-shellcheck.sh || failed+=(shellcheck)
fi

if [[ "${VERIFY_SHELL_FORMAT:-true}" == "true" ]]; then
  echo "[*] Verifying shell format..."
  "${ROOT_DIR}"/hack/verify-shell-format.sh || failed+=(shell-format)
fi

if [[ "${VERIFY_YAML_FORMAT:-true}" == "true" ]]; then
  echo "[*] Verifying yaml format..."
  "${ROOT_DIR}"/hack/verify-yaml-format.sh || failed+=(yaml-format)
fi

if [[ "${VERIFY_SPELLING:-true}" == "true" ]]; then
  echo "[*] Verifying spelling..."
  "${ROOT_DIR}"/hack/verify-spelling.sh || failed+=(spelling)
fi

# exit based on verify scripts
if [[ "${#failed[@]}" != 0 ]]; then
  echo "Verify failed for: ${failed[*]}"
  exit 1
fi
