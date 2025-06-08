#!/bin/sh -eux

TESTPROC="$1"
TESTROOT="$2"
OUTPUT=$(mktemp "/tmp/${TESTPROC}.XXXXXX")

cleanup() {
    rm -f "${OUTPUT}"
}

trap "cleanup" EXIT

_GOARCH=$(go env GOARCH)

if [ ! -x "${TESTROOT}/test_${TESTPROC}" ]; then
    echo "${TESTROOT}/test_${TESTPROC} not found" >&2
    exit 1
fi

"${TESTROOT}/test_${TESTPROC}" >"${OUTPUT}" 2>&1 &

pid=$!

sleep 1

"${TESTROOT}/../bin/watchmaker_linux_${_GOARCH}" --faketime '2021-01-01' --pid "$pid"

wait

cat "${OUTPUT}"

grep -l -- "2021" "${OUTPUT}"
