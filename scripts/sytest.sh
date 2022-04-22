#!/usr/bin/env bash

srcDir=${1}

logsFolder="sqlite"
if [[ ${POSTGRES} ]]; then
  logsFolder="postgres"
fi

if [[ -n ${API} ]]; then
  logsFolder="${logsFolder}_full_api"
fi

output="${logsFolder}.duration"

echo "${logsFolder} - Start: $(date)" > "${output}"

SECONDS=0
docker run --rm \
  -e BUILDKITE_LABEL="local_sytest" \
  -e POSTGRES=${POSTGRES:-0} \
  -e API=${API:-0} \
  -e DENDRITE_TRACE_SQL=${TRACE_SQL:-0} \
  -e DENDRITE_TRACE_HTTP=${TRACE_HTTP:-0} \
  -e DENDRITE_TRACE_INTERNAL=${TRACE_INTERNAL:-0} \
  -v "${srcDir}":/src/:ro \
  -v /tmp/sytest/go-build:/root/.cache/go-build \
  -v "${srcDir}/sytest_${logsFolder}/jetstream":/root/jetstream/ \
  -v "${srcDir}/sytest_${logsFolder}":/logs/ \
  -v "${GOPATH}":/gopath \
  -v "${HOME}/Dev/perl/sytest/":/sytest:ro \
  matrixdotorg/sytest-dendrite ${TEST}
duration=${SECONDS}
echo "${logsFolder} - End: $(date)" >> "${output}"
echo "${logsFolder} - $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed." >> "${output}"
