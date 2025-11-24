#!/usr/bin/env bash
set -euo pipefail

# Runs NIST SP800-90B reference binaries against sample datasets and compares
# the results with the Go/CGO implementation.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${ROOT}/build/90b-validation"
REF_DIR="${BUILD_DIR}/ref"
BIN_DIR="${BUILD_DIR}/bin"
SAMPLES_DIR="${BUILD_DIR}/samples"

MIN_SAMPLES=1000000
# Optional runtime controls:
#   DATASETS="file1.bin file2.bin"   only process listed datasets (filenames)
#   SKIP_NONIID=1                    skip Non-IID runs
#   FORCE=1                          re-run even if cached JSON exists

mkdir -p "${BUILD_DIR}" "${BIN_DIR}" "${SAMPLES_DIR}"

log() { echo "[$(date +'%H:%M:%S')] $*"; }

clone_ref_repo() {
  if [[ -d "${REF_DIR}/.git" ]]; then
    log "Reusing existing reference repo at ${REF_DIR}"
    return
  fi

  log "Cloning NIST SP800-90B reference implementation..."
  git clone --depth=1 https://github.com/usnistgov/SP800-90B_EntropyAssessment.git "${REF_DIR}" >/dev/null 2>&1 || {
    log "Clone failed; falling back to local internal/nist sources"
    mkdir -p "${REF_DIR}"
    rsync -a --delete "${ROOT}/internal/nist/" "${REF_DIR}/"
  }
}

build_ref_binaries() {
  if [[ -x "${BIN_DIR}/ea_iid_ref" && -x "${BIN_DIR}/ea_non_iid_ref" ]]; then
    log "Reusing existing reference binaries in ${BIN_DIR}"
    return
  fi

  clone_ref_repo

  local cpp="${REF_DIR}/cpp"
  local json_inc=""
  if [[ -d /usr/include/jsoncpp ]]; then
    json_inc="-I/usr/include/jsoncpp"
  elif [[ -d /usr/include/json ]]; then
    json_inc="-I/usr/include/json"
  fi

  local includes="-I${cpp} -I${REF_DIR}/wrapper ${json_inc}"
  local libs="-lbz2 -ldivsufsort -ldivsufsort64 -ljsoncpp -lmpfr -lgmp -lcrypto -lgomp -lstdc++ -lm -pthread"

  log "Building reference binaries..."
  g++ -std=c++11 -fopenmp ${includes} "${cpp}/iid_main.cpp" -o "${BIN_DIR}/ea_iid_ref" ${libs}
  g++ -std=c++11 -fopenmp ${includes} "${cpp}/non_iid_main.cpp" -o "${BIN_DIR}/ea_non_iid_ref" ${libs}
}

copy_samples() {
  shopt -s nullglob
  local existing=("${SAMPLES_DIR}"/*.bin)
  if [[ ${#existing[@]} -gt 0 ]]; then
    log "Samples already present in ${SAMPLES_DIR}"
    return
  fi
  log "Copying sample datasets to ${SAMPLES_DIR}"
  cp "${ROOT}"/internal/nist/bin/*.bin "${SAMPLES_DIR}/"
}

parse_min_entropy() {
  python3 - "$1" "$2" <<'PY'
import json,sys
path=sys.argv[1]
bits=int(sys.argv[2])

def coerce_float(x):
    try:
        return float(x)
    except Exception:
        return None

def extract_candidates(obj):
    vals=[]
    if not isinstance(obj, dict):
        return vals
    # Direct fields
    for key in ("hAssessed","h_assessed","hOriginal","h_original","min_entropy","retMinEntropy"):
        if key in obj:
            v=coerce_float(obj[key])
            if v is not None:
                vals.append(v)
    # hBitstring needs scaling by bits/symbol to be comparable
    for key in ("hBitstring","h_bitstring"):
        if key in obj:
            v=coerce_float(obj[key])
            if v is not None and bits>0:
                vals.append(v*bits)
    return vals

try:
    with open(path,"r") as f:
        data=json.load(f)
except Exception:
    print("n/a"); sys.exit(0)

candidates=[]
if isinstance(data, dict):
    candidates.extend(extract_candidates(data))
    tcs=data.get("testCases")
    if isinstance(tcs, list):
        for tc in tcs:
            candidates.extend(extract_candidates(tc))

if candidates:
    print(min(candidates))
else:
    print("n/a")
PY
}

ensure_go_binaries() {
  if [[ ! -x "${ROOT}/build/ea_tool" ]]; then
    log "Building Go CLI (ea_tool)..."
    (cd "${ROOT}" && make build-go >/dev/null)
  fi
}

run_for_file() {
  local file="$1"
  local bits="$2"
  local base
  base=$(basename "${file}")

  # Reference IID
  if [[ "${FORCE:-0}" -eq 1 || ! -f "${BUILD_DIR}/ref_iid_${base}.json" ]]; then
    "${BIN_DIR}/ea_iid_ref" -i -o "${BUILD_DIR}/ref_iid_${base}.json" "${file}" "${bits}" >/dev/null 2>&1 || true
  fi
  # Reference Non-IID
  if [[ "${SKIP_NONIID:-0}" -ne 1 ]]; then
    if [[ "${FORCE:-0}" -eq 1 || ! -f "${BUILD_DIR}/ref_non_iid_${base}.json" ]]; then
      "${BIN_DIR}/ea_non_iid_ref" -i -o "${BUILD_DIR}/ref_non_iid_${base}.json" "${file}" "${bits}" >/dev/null 2>&1 || true
    fi
  fi

  # Go
  if [[ "${FORCE:-0}" -eq 1 || ! -f "${BUILD_DIR}/go_iid_${base}.json" ]]; then
    "${ROOT}/build/ea_tool" -iid -bits "${bits}" -output "${BUILD_DIR}/go_iid_${base}.json" "${file}" >/dev/null 2>&1 || true
  fi
  if [[ "${SKIP_NONIID:-0}" -ne 1 ]]; then
    if [[ "${FORCE:-0}" -eq 1 || ! -f "${BUILD_DIR}/go_non_iid_${base}.json" ]]; then
      "${ROOT}/build/ea_tool" -non-iid -bits "${bits}" -output "${BUILD_DIR}/go_non_iid_${base}.json" "${file}" >/dev/null 2>&1 || true
    fi
  fi

  local ref_iid go_iid ref_non go_non
  ref_iid=$(parse_min_entropy "${BUILD_DIR}/ref_iid_${base}.json" "${bits}")
  go_iid=$(parse_min_entropy "${BUILD_DIR}/go_iid_${base}.json" "${bits}")
  ref_non="n/a"
  go_non="n/a"
  if [[ "${SKIP_NONIID:-0}" -ne 1 ]]; then
    ref_non=$(parse_min_entropy "${BUILD_DIR}/ref_non_iid_${base}.json" "${bits}")
    go_non=$(parse_min_entropy "${BUILD_DIR}/go_non_iid_${base}.json" "${bits}")
  fi

  printf "%-20s %-4s %-12s %-12s %-12s %-12s\n" "${base}" "${bits}" "${ref_iid}" "${go_iid}" "${ref_non}" "${go_non}"
}

main() {
  build_ref_binaries
  copy_samples
  ensure_go_binaries

  printf "%s\n" "Dataset              Bits Ref_IID      Go_IID       Ref_NonIID   Go_NonIID"
  printf "%s\n" "-------------------- ---- ------------ ------------ ------------ ------------"

  shopt -s nullglob
  # Optional dataset selection via DATASETS env var (space-separated basenames)
  declare -A allow=()
  if [[ -n "${DATASETS:-}" ]]; then
    for d in ${DATASETS}; do
      allow["${d}"]=1
    done
  fi

  for f in "${SAMPLES_DIR}"/*.bin; do
    if [[ ${#allow[@]} -gt 0 ]]; then
      b=$(basename "$f")
      [[ -n "${allow[$b]:-}" ]] || continue
    fi

    size=$(stat -c%s "$f")
    if [[ "${size}" -lt "${MIN_SAMPLES}" ]]; then
      log "Skipping ${f} (size ${size} < ${MIN_SAMPLES})"
      continue
    fi
    bits=8
    case "$f" in
      *biased-random-bits*|*truerand_1bit*) bits=1 ;;
      *1bit*) bits=1 ;;
      *4bit*) bits=4 ;;
      *8bit*) bits=8 ;;
      *biased-random-bytes*) bits=8 ;;
    esac
    run_for_file "$f" "$bits"
  done
}

main "$@"
