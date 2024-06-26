#!/usr/bin/env bash

set -e

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 VERSION" >&2
  exit 1
fi

ARCH=$(go env GOARCH)
VERSION="${1}-${ARCH}"
DOCKERFILE="Dockerfile-release.${ARCH}"

if [ -z "${BINARYDIR}" ]; then
  RELEASE="etcd-${1}"-$(go env GOOS)-$(go env GOARCH)
  BINARYDIR="${RELEASE}"
  TARFILE="${RELEASE}.tar.gz"
  TARURL="https://github.com/etcd-io/etcd/releases/download/${1}/${TARFILE}"
  if ! curl -f -L -o "${TARFILE}" "${TARURL}" ; then
    echo "Failed to download ${TARURL}."
    exit 1
  fi
  tar -zvxf "${TARFILE}"
fi

BINARYDIR=${BINARYDIR:-.}
BUILDDIR=${BUILDDIR:-.}

IMAGEDIR=${BUILDDIR}/image-docker

mkdir -p "${IMAGEDIR}"/var/etcd
mkdir -p "${IMAGEDIR}"/var/lib/etcd
cp "${BINARYDIR}"/etcd "${BINARYDIR}"/etcdctl "${BINARYDIR}"/etcdutl "${IMAGEDIR}"

cp ./nsswitch.conf "${IMAGEDIR}"

cat ./"${DOCKERFILE}" > "${IMAGEDIR}"/Dockerfile

if [ -z "$TAG" ]; then
    docker build -t "gcr.io/etcd-development/etcd:${VERSION}" "${IMAGEDIR}"
    docker build -t "quay.io/coreos/etcd:${VERSION}" "${IMAGEDIR}"
else
    docker build -t "${TAG}:${VERSION}" "${IMAGEDIR}"
fi
