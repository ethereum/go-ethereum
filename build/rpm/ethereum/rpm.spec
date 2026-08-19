# Go binaries are linked with --build-id=none and --strip-all (see buildFlags in
# build/ci.go), so there is nothing for the debuginfo machinery to extract.
%global debug_package %{nil}
%global _build_id_links none
%undefine _missing_build_ids_terminate_build

Name:           {{.Name}}
Version:        {{.Version}}
Release:        {{.Release}}
Summary:        Meta-package to install geth and other tools
License:        GPL-3.0-or-later AND LGPL-3.0-or-later
URL:            https://geth.ethereum.org
Source0:        {{.SourceName}}

BuildRequires:  golang >= 1.24
BuildRequires:  gcc
{{range .Executables}}Requires:       {{$.ExeName .}} = %{version}-%{release}
{{end}}
%description
Meta-package to install geth and other tools
{{range .Executables}}
%package -n {{$.ExeName .}}
Summary:        {{.Description}}
{{- if $.ExeConflicts .}}
Conflicts:      {{$.ExeConflicts .}}
{{- end}}

%description -n {{$.ExeName .}}
{{.Description}}
{{end}}
%prep
%autosetup -n {{.Name}}-{{.Version}}

%build
# COPR builders have no network access, so all module downloads are faked out
# with the cache shipped inside the source tarball. GOPROXY=off makes a missing
# module fail loudly instead of hanging on a network timeout.
export GOPATH=%{_builddir}/gopath
export GOCACHE=%{_builddir}/go-build
export GOPROXY=off
export GOSUMDB=off
export GOTOOLCHAIN=local

mkdir -p $GOPATH/pkg
mv .mod $GOPATH/pkg/mod

go run build/ci.go install -git-commit={{.Env.Commit}} -git-branch={{.Env.Branch}} -git-tag={{.Env.Tag}} -buildnum={{.Env.Buildnum}} -pull-request={{.Env.IsPullRequest}}

%install
install -d %{buildroot}%{_bindir}
{{range .Executables}}install -p -m 0755 build/bin/{{.BinaryName}} %{buildroot}%{_bindir}/{{.BinaryName}}
{{- if eq .BinaryName "geth"}}
install -D -p -m 0644 build/completions/bash/geth %{buildroot}%{_datadir}/bash-completion/completions/geth
install -D -p -m 0644 build/completions/zsh/_geth %{buildroot}%{_datadir}/zsh/site-functions/_geth
{{- end}}
{{end}}
%files
{{range .Executables}}
%files -n {{$.ExeName .}}
%license COPYING COPYING.LESSER
%doc AUTHORS
%{_bindir}/{{.BinaryName}}
{{- if eq .BinaryName "geth"}}
%{_datadir}/bash-completion/completions/geth
%{_datadir}/zsh/site-functions/_geth
{{- end}}
{{end}}
%changelog
* {{.Time}} {{.Author}} - {{.Version}}-{{.Release}}
- git build of {{.Env.Commit}}
