#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: scripts/newmodule.sh <context> <service>" >&2
  exit 1
fi

context="$1"
service="$2"
module_path="contexts/${context}/${service}"

if [[ -e "${module_path}" ]]; then
  echo "module already exists: ${module_path}" >&2
  exit 1
fi

mkdir -p \
  "${module_path}/domain" \
  "${module_path}/application" \
  "${module_path}/ports" \
  "${module_path}/adapters" \
  "${module_path}/transport"

title="$(echo "${service}" | sed -E 's/(^|-)([a-z])/\U\2/g' | sed -E 's/([A-Z])/\1 /g' | xargs)"

cat > "${module_path}/README.md" <<EOF
# ${title}

Module scaffold for Solomon monolith.

## Structure
- domain/: entities, value objects, domain services, invariants
- application/: use cases, command/query handlers, orchestration
- ports/: repository, event, and client interfaces
- adapters/: DB, HTTP/gRPC, event bus, cache implementations
- transport/: module-private transport DTOs and event payload mappers
EOF

echo "Created module scaffold at ${module_path}"
