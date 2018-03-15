pkg_name="habitat-operator"
pkg_origin="habitat"
pkg_version=$(cat "$PLAN_CONTEXT/../VERSION")
pkg_description="A Kubernetes operator for Habitat services"
pkg_upstream_url="https://github.com/habitat-sh/habitat-operator"
pkg_license=('Apache-2.0')
pkg_maintainer="The Habitat Maintainers <humans@habitat.sh>"
pkg_bin_dirs=(bin)
scaffolding_go_base_path=github.com/habitat-sh
pkg_scaffolding=core/scaffolding-go
pkg_svc_run="${pkg_name}"

do_build() {
  pushd "$scaffolding_go_pkg_path" >/dev/null
  make -e BIN_PATH="${scaffolding_go_gopath}/bin/habitat-operator" linux
  popd >/dev/null
}

do_install() {
  cp -r "${scaffolding_go_gopath}/bin" "${pkg_prefix}/${bin}"
}
